package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const dockerImage = "churlee/vpn-node-builder"

type cfg struct {
	ipType     string // "dynamic" | "static"
	duckDomain string
	duckToken  string
	staticIP   string
	apiKey     string
	sshPass    string
}

func main() {
	printBanner()
	checkDocker()
	checkWireGuard()

	c := runWizard()
	pullDockerImage()
	runBuildContainer(c)
	printCompletion(c)
}

func printBanner() {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 2)
	fmt.Println()
	fmt.Println(style.Render("  RAM-Only VPN Node — Setup Wizard  "))
	fmt.Println()
}

func checkWireGuard() {
	if runtime.GOOS != "darwin" {
		return
	}
	if _, err := exec.LookPath("wg-quick"); err == nil {
		return
	}
	fmt.Print("wg-quick not found — required for the VPN client on macOS... ")
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("installing via Homebrew...")
		cmd := exec.Command("brew", "install", "wireguard-tools")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println()
			fmt.Println("brew install failed. Run manually: brew install wireguard-tools")
			os.Exit(1)
		}
		fmt.Println("wg-quick installed.")
	} else {
		fmt.Println("Homebrew not found.")
		fmt.Println()
		fmt.Println("Install Homebrew first:")
		fmt.Println(`  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
		fmt.Println("Then re-run the installer.")
		os.Exit(1)
	}
	fmt.Println()
}

func checkDocker() {
	fmt.Print("Checking Docker... ")
	if err := exec.Command("docker", "info").Run(); err != nil {
		fmt.Println("not found or not running.")
		fmt.Println()
		fmt.Println("Install Docker: https://docs.docker.com/get-docker/")
		os.Exit(1)
	}
	fmt.Println("ok")
	fmt.Println()
}

func runWizard() cfg {
	var c cfg
	c.ipType = "dynamic"
	confirmed := false

	form := huh.NewForm(
		// Step 1: IP type
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How will clients find your Pi?").
				Options(
					huh.NewOption("Dynamic IP via DuckDNS (recommended)", "dynamic"),
					huh.NewOption("Static public IP address", "static"),
				).
				Value(&c.ipType),
		),

		// Step 2a: DuckDNS credentials (shown only for dynamic)
		huh.NewGroup(
			huh.NewInput().
				Title("DuckDNS Domain").
				Description("Your subdomain without .duckdns.org  e.g. myvpn").
				Placeholder("myvpn").
				Value(&c.duckDomain).
				Validate(notEmpty("DuckDNS domain")),
			huh.NewInput().
				Title("DuckDNS Token").
				Description("Found on duckdns.org after logging in").
				EchoMode(huh.EchoModePassword).
				Value(&c.duckToken).
				Validate(notEmpty("DuckDNS token")),
		).WithHideFunc(func() bool { return c.ipType != "dynamic" }),

		// Step 2b: Static IP (shown only for static)
		huh.NewGroup(
			huh.NewInput().
				Title("Static IP Address").
				Description("Public IP your Pi is reachable at").
				Placeholder("1.2.3.4").
				Value(&c.staticIP).
				Validate(notEmpty("static IP")),
		).WithHideFunc(func() bool { return c.ipType != "static" }),

		// Step 3: API key
		huh.NewGroup(
			huh.NewInput().
				Title("API Key").
				Description("Clients enter this to authenticate — make it strong").
				EchoMode(huh.EchoModePassword).
				Value(&c.apiKey).
				Validate(notEmpty("API key")),
		),

		// Step 4: SSH password
		huh.NewGroup(
			huh.NewInput().
				Title("SSH Password (optional)").
				Description("Root password for Pi SSH access. Leave blank to skip for now.").
				EchoMode(huh.EchoModePassword).
				Value(&c.sshPass),
		),

		// Step 5: Confirm
		huh.NewGroup(
			huh.NewConfirm().
				Title("Start build?").
				DescriptionFunc(func() string { return buildSummary(&c) }, &c).
				Value(&confirmed),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		fmt.Println("\nCancelled.")
		os.Exit(0)
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	return c
}

func notEmpty(field string) func(string) error {
	return func(v string) error {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

func buildSummary(c *cfg) string {
	var sb strings.Builder
	if c.ipType == "dynamic" {
		fmt.Fprintf(&sb, "Mode:   DuckDNS\nDomain: %s.duckdns.org", c.duckDomain)
	} else {
		fmt.Fprintf(&sb, "Mode: Static IP\nAddr: %s", c.staticIP)
	}
	return sb.String()
}

func pullDockerImage() {
	step("Pulling build environment from Docker Hub...")
	cmd := exec.Command("docker", "pull", dockerImage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatalf("docker pull failed: %v", err)
	}
}

func runBuildContainer(c cfg) {
	step("Running build container (this may take a few minutes on first run)...")

	exe, err := os.Executable()
	if err != nil {
		fatalf("could not resolve executable path: %v", err)
	}
	outputDir := filepath.Join(filepath.Dir(exe), "vpn-node-output")
	_ = os.RemoveAll(outputDir)
	if err := os.MkdirAll(outputDir, 0777); err != nil {
		fatalf("create output dir: %v", err)
	}
	_ = os.Chmod(outputDir, 0777)

	args := []string{
		"run", "--rm",
		"-v", outputDir + ":/output",
		"-e", "CLIENT_OS=" + runtime.GOOS,
		"-e", "NODE_API_KEY=" + c.apiKey,
	}
	if c.ipType == "dynamic" {
		args = append(args, "-e", "DUCKDNS_TOKEN="+c.duckToken)
		args = append(args, "-e", "DUCKDNS_DOMAIN="+c.duckDomain)
	} else {
		args = append(args, "-e", "STATIC_IP="+c.staticIP)
	}
	if c.sshPass != "" {
		args = append(args, "-e", "SSH_PASS="+c.sshPass)
	}
	args = append(args, dockerImage)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatalf("build failed: %v", err)
	}
}

func printCompletion(c cfg) {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("76"))
	key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))

	fmt.Println()
	fmt.Println(title.Render("✓ Build complete"))
	fmt.Println()
	fmt.Println("vpn-node-output/pi-flash/  — copy ALL files to the root of a FAT32 SD card and boot your Pi.")
	fmt.Println()

	fmt.Println(key.Render("PORT FORWARDING"))
	fmt.Println("  Forward these on your router to your Pi's LAN IP:")
	fmt.Println("  • TCP 8080  — VPN API (client connections)")
	fmt.Println("  • UDP 51820 — WireGuard tunnel")
	fmt.Println()

	fmt.Println(key.Render("COMMON ISSUES"))
	fmt.Println()

	fmt.Println("  DNS not resolving?")
	fmt.Println("    Set your router's DNS server to 1.1.1.1 or 8.8.8.8.")
	if c.ipType == "dynamic" {
		fmt.Printf("    Check propagation: nslookup %s.duckdns.org\n", c.duckDomain)
		fmt.Println("    DuckDNS updates on Pi boot and every hour — wait a minute and retry.")
	}
	fmt.Println()

	fmt.Println("  Can't connect from the client?")
	fmt.Println("    Confirm port forwarding is working:")
	if c.ipType == "dynamic" {
		fmt.Printf("    nmap -p 8080 %s.duckdns.org\n", c.duckDomain)
	} else {
		fmt.Printf("    nmap -p 8080 %s\n", c.staticIP)
	}
	fmt.Println()

	fmt.Println("  Behind CGNAT?")
	fmt.Println("    Your router's WAN IP starts with 100.64.x.x or is another private range.")
	fmt.Println("    This means your ISP controls an outer NAT — port forwarding won't reach you.")
	fmt.Println("    Fix: ask ISP for a public IP, use a VPS relay, or switch providers.")
	fmt.Println()

	fmt.Println("  SSH into the Pi for debugging:")
	fmt.Println("    ssh root@<LAN-IP>   (find it in your router's DHCP lease table)")
	fmt.Println()
}

func step(msg string) {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	fmt.Println(s.Render("==> " + msg))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

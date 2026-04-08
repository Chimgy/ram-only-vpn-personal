export namespace main {
	
	export class ConnectResult {
	    ok: boolean;
	    tunnelIP: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.tunnelIP = source["tunnelIP"];
	        this.error = source["error"];
	    }
	}
	export class StatusResult {
	    connected: boolean;
	    tunnelIP: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.tunnelIP = source["tunnelIP"];
	    }
	}

}


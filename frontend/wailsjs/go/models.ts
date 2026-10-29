export namespace ai {
	
	export class AIRequest {
	    provider: string;
	    model: string;
	    prompt: string;
	    userNotes: string;
	    telemetry: string;
	
	    static createFrom(source: any = {}) {
	        return new AIRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.userNotes = source["userNotes"];
	        this.telemetry = source["telemetry"];
	    }
	}
	export class AIResponse {
	    text: string;
	    inputTokens: number;
	    outputTokens: number;
	    costUsd: number;
	
	    static createFrom(source: any = {}) {
	        return new AIResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.costUsd = source["costUsd"];
	    }
	}
	export class ScrubMatch {
	    type: string;
	    original: string;
	    replacement: string;
	
	    static createFrom(source: any = {}) {
	        return new ScrubMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.original = source["original"];
	        this.replacement = source["replacement"];
	    }
	}

}

export namespace db {
	
	export class DiscoveredLog {
	    id: number;
	    filePath: string;
	    appName: string;
	    fileSize: number;
	    // Go type: time
	    lastModified: any;
	    format: string;
	    isFavorite: boolean;
	    tag: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filePath = source["filePath"];
	        this.appName = source["appName"];
	        this.fileSize = source["fileSize"];
	        this.lastModified = this.convertValues(source["lastModified"], null);
	        this.format = source["format"];
	        this.isFavorite = source["isFavorite"];
	        this.tag = source["tag"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Store {
	
	
	    static createFrom(source: any = {}) {
	        return new Store(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace main {
	
	export class ScrubResult {
	    text: string;
	    matches: ai.ScrubMatch[];
	
	    static createFrom(source: any = {}) {
	        return new ScrubResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.matches = this.convertValues(source["matches"], ai.ScrubMatch);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}


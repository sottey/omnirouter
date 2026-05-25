export namespace main {
	
	export class Target {
	    name: string;
	    type: string;
	    app?: string;
	    url?: string;
	    description?: string;
	    shortcut?: string;
	    sendMode: string;
	    startupDelayMs: number;
	    provider?: string;
	    model?: string;
	    apiKeyEnv?: string;
	    systemPrompt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.app = source["app"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.shortcut = source["shortcut"];
	        this.sendMode = source["sendMode"];
	        this.startupDelayMs = source["startupDelayMs"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.systemPrompt = source["systemPrompt"];
	    }
	}
	export class Router {
	    provider: string;
	    model: string;
	    apiKeyEnv: string;
	    systemPrompt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Router(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.systemPrompt = source["systemPrompt"];
	    }
	}
	export class Config {
	    showWindowOnStartup?: boolean;
	    defaultTarget?: string;
	    autoHideAfterSend?: boolean;
	    hotkeyMode?: string;
	    launcherWindowWidth?: number;
	    launcherWindowHeight?: number;
	    windowWidth?: number;
	    windowHeight?: number;
	    windowX?: number;
	    windowY?: number;
	    router?: Router;
	    targets: Target[];
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.showWindowOnStartup = source["showWindowOnStartup"];
	        this.defaultTarget = source["defaultTarget"];
	        this.autoHideAfterSend = source["autoHideAfterSend"];
	        this.hotkeyMode = source["hotkeyMode"];
	        this.launcherWindowWidth = source["launcherWindowWidth"];
	        this.launcherWindowHeight = source["launcherWindowHeight"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.windowX = source["windowX"];
	        this.windowY = source["windowY"];
	        this.router = this.convertValues(source["router"], Router);
	        this.targets = this.convertValues(source["targets"], Target);
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
	
	export class SendPromptResult {
	    targetName: string;
	    responseText?: string;
	    isApi: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SendPromptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetName = source["targetName"];
	        this.responseText = source["responseText"];
	        this.isApi = source["isApi"];
	    }
	}

}


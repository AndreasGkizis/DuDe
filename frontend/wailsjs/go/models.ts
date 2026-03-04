export namespace models {
	
	export class ExecutionParams {
	    directories: string[];
	    useCache: boolean;
	    cacheDir: string;
	    resultsDir: string;
	    paranoidMode: boolean;
	    cpus: number;
	    bufSize: number;
	    debugMode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecutionParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.directories = source["directories"];
	        this.useCache = source["useCache"];
	        this.cacheDir = source["cacheDir"];
	        this.resultsDir = source["resultsDir"];
	        this.paranoidMode = source["paranoidMode"];
	        this.cpus = source["cpus"];
	        this.bufSize = source["bufSize"];
	        this.debugMode = source["debugMode"];
	    }
	}
	export class FileHash {
	    FileName: string;
	    FilePath: string;
	    Hash: string;
	    ModTime: string;
	    FileSize: number;
	    DuplicatesFound: FileHash[];
	
	    static createFrom(source: any = {}) {
	        return new FileHash(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FileName = source["FileName"];
	        this.FilePath = source["FilePath"];
	        this.Hash = source["Hash"];
	        this.ModTime = source["ModTime"];
	        this.FileSize = source["FileSize"];
	        this.DuplicatesFound = this.convertValues(source["DuplicatesFound"], FileHash);
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


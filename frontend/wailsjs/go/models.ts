export namespace models {
	
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
	export class DuplicateResult {
	    groups: FileHash[];
	    reportPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groups = this.convertValues(source["groups"], FileHash);
	        this.reportPath = source["reportPath"];
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
	export class ExecutionParams {
	    mode: number;
	    directories: string[];
	    findMissingSourceDir: string;
	    findMissingTargetDir: string;
	    useCache: boolean;
	    cacheDir: string;
	    resultsDir: string;
	    paranoidMode: boolean;
	    workerCount: number;
	    bufferSizeBytes: number;
	    debugMode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecutionParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.directories = source["directories"];
	        this.findMissingSourceDir = source["findMissingSourceDir"];
	        this.findMissingTargetDir = source["findMissingTargetDir"];
	        this.useCache = source["useCache"];
	        this.cacheDir = source["cacheDir"];
	        this.resultsDir = source["resultsDir"];
	        this.paranoidMode = source["paranoidMode"];
	        this.workerCount = source["workerCount"];
	        this.bufferSizeBytes = source["bufferSizeBytes"];
	        this.debugMode = source["debugMode"];
	    }
	}
	
	export class MissingSourceFile {
	    filename: string;
	    fullPath: string;
	
	    static createFrom(source: any = {}) {
	        return new MissingSourceFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.fullPath = source["fullPath"];
	    }
	}
	export class MissingContentResult {
	    totalSourceFiles: number;
	    missingFiles: MissingSourceFile[];
	    reportPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new MissingContentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalSourceFiles = source["totalSourceFiles"];
	        this.missingFiles = this.convertValues(source["missingFiles"], MissingSourceFile);
	        this.reportPath = source["reportPath"];
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


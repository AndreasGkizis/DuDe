export namespace models {
	
	export class ExecutionParams {
	    sourceDir: string;
	    targetDir: string;
	    cacheDir: string;
	    resultsDir: string;
	    paranoidMode: boolean;
	    cpus: number;
	    bufSize: number;
	    dualFolderModeEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExecutionParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceDir = source["sourceDir"];
	        this.targetDir = source["targetDir"];
	        this.cacheDir = source["cacheDir"];
	        this.resultsDir = source["resultsDir"];
	        this.paranoidMode = source["paranoidMode"];
	        this.cpus = source["cpus"];
	        this.bufSize = source["bufSize"];
	        this.dualFolderModeEnabled = source["dualFolderModeEnabled"];
	    }
	}

}


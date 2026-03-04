/**
 * Frontend model for a single duplicate entry (original or duplicate file).
 * Mapped from the backend models.FileHash (PascalCase fields).
 */
export class FrontEnd_DuplicateFile {
    /**
     * @param {string} fileName
     * @param {string} filePath
     */
    constructor(fileName, filePath) {
        this.fileName = fileName;
        this.filePath = filePath;
    }
}

/**
 * Frontend model representing one group of duplicates:
 * the original file and all files considered its duplicates.
 * Mapped from the backend models.FileHash (PascalCase fields).
 */
export class FrontEnd_DuplicateGroup {
    /**
     * @param {string} fileName
     * @param {string} filePath
     * @param {FrontEnd_DuplicateFile[]} duplicates
     */
    constructor(fileName, filePath, duplicates) {
        this.fileName = fileName;
        this.filePath = filePath;
        /** @type {FrontEnd_DuplicateFile[]} */
        this.duplicates = duplicates;
    }

    /**
     * Maps a raw backend FileHash object (as returned by GetResults()) to a FrontEnd_DuplicateGroup.
     * @param {import('../wailsjs/go/models').models.FileHash} fh
     * @returns {FrontEnd_DuplicateGroup}
     */
    static fromFileHash(fh) {
        const duplicates = (fh.DuplicatesFound || []).map(
            d => new FrontEnd_DuplicateFile(d.FileName, d.FilePath)
        );
        return new FrontEnd_DuplicateGroup(fh.FileName, fh.FilePath, duplicates);
    }
}

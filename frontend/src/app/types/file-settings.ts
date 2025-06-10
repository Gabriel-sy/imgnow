export interface FileSettings {
  expiresIn?: Date;
  deletesAfterDownload?: boolean;
  downloadsForDeletion?: number;
  deletesAfterVizualizations?: boolean;
  vizualizationsForDeletion?: number;
  password?: string;
}

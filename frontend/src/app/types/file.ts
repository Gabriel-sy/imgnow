export type File = {
  id: number;
  customUrl: string;
  path?: string;
  originalName: string;
  size: number;
  type: string;
  createdAt: Date;
  status: string;
  vizualizations: number;
  downloads: number;
  deletesAfterDownload: boolean;
  deletedAt?: Date;
  downloadsForDeletion?: number;
  deletesAfterVizualizations: boolean;
  vizualizationsForDeletion?: number;
  lastVizualization?: Date;
  expiresIn?: Date;
}
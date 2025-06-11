import { HttpClient, HttpEvent, HttpEventType } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { FileSettings } from '../types/file-settings';
import { File as FileType } from '../types/file'; // Import FileType

interface UploadResponse {
  message: string;
  status: string;
  customUrl: string;
  statusUrl: string;
}

@Injectable({
  providedIn: 'root',
})
export class FileService {
  private readonly API_URL = environment.apiUrl + '/file';

  constructor(private http: HttpClient) {}

  uploadFile(file: File, customUrl: string): Observable<UploadResponse> {
    const formData = new FormData();
    formData.append('file', file);

    return this.http.post<UploadResponse>(this.API_URL + '/upload', formData, {
      params: { customUrl },
    });
  }

  setFileSettings(customUrl: string, settings: FileSettings): Observable<any> {
    return this.http.put(
      this.API_URL + '/' + customUrl + '/settings',
      settings
    );
  }

  getFileWithPassword(
    customUrl: string,
    password: string
  ): Observable<{ path: string; requiresPassword?: boolean }> {
    return this.http.post<{ path: string; requiresPassword?: boolean }>(
      this.API_URL + '/' + customUrl,
      { password }
    );
  }

  getFileWithoutPassword(customUrl: string): Observable<{ path: string; requiresPassword?: boolean }> {
    return this.http.get<{ path: string; requiresPassword?: boolean }>(
      this.API_URL + '/' + customUrl
    );
  }

  getFileStatus(customUrl: string): Observable<{ status: string }> {
    return this.http.get<{ status: string }>(
      this.API_URL + '/' + customUrl + '/status'
    );
  }

  getFileInfo(customUrl: string): Observable<FileType> {
    return this.http.get<FileType>(`${this.API_URL}/${customUrl}/info`);
  }

  addDownload(customUrl: string): Observable<any> {
    return this.http.put(`${this.API_URL}/${customUrl}/addDownload`, {});
  }
}

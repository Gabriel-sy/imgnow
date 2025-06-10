import { HttpClient, HttpEvent, HttpEventType } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { FileSettings } from '../types/file-settings';

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

  getFileLink(customUrl: string): Observable<{ path: string }> {
    return this.http.get<{ path: string }>(this.API_URL + '/' + customUrl);
  }

  getFileStatus(customUrl: string): Observable<{ status: string }> {
    return this.http.get<{ status: string }>(
      this.API_URL + '/' + customUrl + '/status'
    );
  }
}

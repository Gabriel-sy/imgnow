import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { Subscription } from 'rxjs';
import { FormsModule } from '@angular/forms';
import { FileService } from '../../services/file-service';
import { File as FileType } from '../../types/file';

@Component({
  selector: 'app-file-view',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  templateUrl: './file-view.component.html',
  styleUrls: ['./file-view.component.css'],
})
export class FileViewComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private fileService = inject(FileService);
  private sanitizer = inject(DomSanitizer);

  customUrl: string | null = null;
  fileData: FileType | null = null;
  fileContentUrl: SafeResourceUrl | null = null;
  isLoading: boolean = true;
  errorFetchingFile: string | null = null;
  requiresPassword = false;
  password = '';
  invalidPassword = false;
  fileIsPending = false;
  private routeSubscription: Subscription | undefined;
  private fileSubscription: Subscription | undefined;

  ngOnInit(): void {
    this.routeSubscription = this.route.paramMap.subscribe((params) => {
      this.customUrl = params.get('customUrl');
      if (this.customUrl) {
        this.fetchFile(this.customUrl);
      } else {
        this.isLoading = false;
        this.errorFetchingFile = 'No custom URL provided.';
      }
    });
  }

  fetchFile(customUrl: string, password?: string): void {
    this.isLoading = true;
    this.errorFetchingFile = null;
    this.invalidPassword = false;

    if (!password) {
      this.getFileWithoutPassword(customUrl);
    } else {
      this.getFileWithPassword(customUrl, password);
    }
  }

  fetchFileInfo(customUrl: string): void {
    this.fileService.getFileInfo(customUrl).subscribe({
      next: (data) => {
        this.fileData = data;
        this.isLoading = false;
      },
      error: (err) => {
        this.errorFetchingFile =
          err.error?.error || 'Error fetching file details.';
        this.isLoading = false;
      },
    });
  }

  onPasswordSubmit(): void {
    if (this.customUrl && this.password) {
      this.fetchFile(this.customUrl, this.password);
    }
  }

  retry(): void {
    this.fileIsPending = false;
    if (this.customUrl) {
      this.fetchFile(this.customUrl, this.password);
    }
  }

  downloadFile(): void {
    if (this.fileData && this.customUrl && this.fileContentUrl) {
      this.fileService.addDownload(this.customUrl).subscribe();
      const a = document.createElement('a');
      a.href = this.fileContentUrl.toString();
      a.download = this.fileData.originalName;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
    }
  }

  getFileWithPassword(customUrl: string, password: string): void {
    this.fileSubscription = this.fileService
      .getFileWithPassword(customUrl, password)
      .subscribe({
        next: (response) => {
          if (response.requiresPassword) {
            this.requiresPassword = true;
            this.isLoading = false;
          } else if (response.path) {
            this.requiresPassword = false;
            this.fileContentUrl = this.sanitizer.bypassSecurityTrustResourceUrl(
              response.path
            );
            this.fetchFileInfo(customUrl);
          }
        },
        error: (err) => {
          if (err.status === 401) {
            this.invalidPassword = true;
          } else if (err.status === 403) {
            this.requiresPassword = true;
          } else if (err.status === 425) {
            this.fileIsPending = true;
          } else {
            this.errorFetchingFile = err.error?.error || 'Error fetching file.';
          }
          this.isLoading = false;
        },
      });
  }

  getFileWithoutPassword(customUrl: string): void {
    this.fileSubscription = this.fileService
      .getFileWithoutPassword(customUrl)
      .subscribe({
        next: (response) => {
          if (response.requiresPassword) {
            this.requiresPassword = true;
            this.isLoading = false;
          } else if (response.path) {
            this.requiresPassword = false;
            this.fileContentUrl = this.sanitizer.bypassSecurityTrustResourceUrl(
              response.path
            );
            this.fetchFileInfo(customUrl);
          }
        },
        error: (err) => {
          if (err.status === 401) {
            this.invalidPassword = true;
          } else if (err.status === 403) {
            this.requiresPassword = true;
          } else if (err.status === 425) {
            this.fileIsPending = true;
          } else {
            this.errorFetchingFile = err.error?.error || 'Error fetching file.';
          }
          this.isLoading = false;
        },
      });
  }

  ngOnDestroy(): void {
    this.routeSubscription?.unsubscribe();
    this.fileSubscription?.unsubscribe();
  }

  isImage(): boolean {
    return !!this.fileData && this.fileData.type?.startsWith('image/');
  }

  isVideo(): boolean {
    return !!this.fileData && this.fileData.type?.startsWith('video/');
  }
}

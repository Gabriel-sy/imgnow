import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSnackBar } from '@angular/material/snack-bar';
import { FileService } from '../../../services/file-service';
import { firstValueFrom } from 'rxjs';

const MAX_FILE_SIZE = 30 * 1024 * 1024; // 30MB in bytes
const ALLOWED_FILE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'video/mp4',
  'video/webm',
];

@Component({
  selector: 'app-upload-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatIconModule,
    MatInputModule,
    MatFormFieldModule,
    MatProgressBarModule,
    MatDialogModule,
  ],
  templateUrl: './upload-dialog.component.html',
  styleUrls: ['./upload-dialog.component.css'],
})
export class UploadDialogComponent implements OnInit {
  customUrl: string = '';
  selectedFile: File | null = null;
  isDragging = false;
  uploadProgress = 0;
  isUploading = false;
  errorMessage: string = '';

  constructor(
    public dialogRef: MatDialogRef<UploadDialogComponent>,
    private fileService: FileService,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit(): void {}

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = true;
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = false;
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    event.stopPropagation();
    this.isDragging = false;

    const files = event.dataTransfer?.files;
    if (files && files.length > 0) {
      this.handleFile(files[0]);
    }
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.handleFile(input.files[0]);
    }
  }

  private handleFile(file: File): void {
    this.errorMessage = '';

    if (!ALLOWED_FILE_TYPES.includes(file.type)) {
      this.errorMessage =
        'Only images (JPEG, PNG, GIF, WEBP) and videos (MP4, WEBM) are allowed.';
      this.showError();
      return;
    }

    if (file.size > MAX_FILE_SIZE) {
      this.errorMessage = 'File size must be less than 30MB.';
      this.showError();
      return;
    }

    this.selectedFile = file;
  }

  private showError(): void {
    this.snackBar.open(this.errorMessage, '', {
      duration: 5000,
      horizontalPosition: 'center',
      verticalPosition: 'bottom',
      panelClass: ['error-snackbar'],
    });
  }

  private showSuccess(message: string): void {
    this.snackBar.open(message, '', {
      duration: 5000,
      horizontalPosition: 'center',
      verticalPosition: 'bottom',
      panelClass: ['success-snackbar'],
    });
  }

  async uploadFile(): Promise<void> {
    if (!this.selectedFile) return;

    this.isUploading = true;
    this.uploadProgress = 0;

    try {
      // Simulate upload progress
      const progressInterval = setInterval(() => {
        if (this.uploadProgress < 90) {
          this.uploadProgress += 10;
        }
      }, 500);

      const response = await firstValueFrom(
        this.fileService.uploadFile(this.selectedFile, this.customUrl)
      );

      clearInterval(progressInterval);
      this.uploadProgress = 100;
      this.showSuccess('File uploaded successfully!');
      this.dialogRef.close({ customUrl: response.customUrl });
    } catch (error) {
      console.error('Upload failed:', error);
      this.errorMessage = 'Upload failed. Please try again.';
      this.showError();
    } finally {
      this.isUploading = false;
    }
  }

  close(): void {
    this.dialogRef.close();
  }

  formatFileSize(bytes: number): string {
    if (bytes === 0) {
      return '0 Bytes';
    }
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
}

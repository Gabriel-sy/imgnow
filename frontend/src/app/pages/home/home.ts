import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatDialog } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { FileService } from '../../services/file-service';
import { UploadDialogComponent } from './upload-dialog/upload-dialog.component';
import { ConfigurationDialogComponent } from './configuration-dialog/configuration-dialog.component';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatButtonModule,
    MatIconModule,
    MatInputModule,
    MatFormFieldModule,
  ],
  templateUrl: './home.html',
  styleUrls: ['./home.css'],
})
export class Home {
  constructor(private dialog: MatDialog, private fileService: FileService) {}

  openUploadDialog(): void {
    const dialogRef = this.dialog.open(UploadDialogComponent, {
      width: '500px',
      panelClass: 'upload-dialog',
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result && result.customUrl) {
        // Open configuration dialog after successful upload
        const configDialogRef = this.dialog.open(ConfigurationDialogComponent, {
          width: '500px',
          data: { customUrl: result.customUrl },
        });

        configDialogRef.afterClosed().subscribe((configResult) => {
          if (configResult) {
            // Update file settings
            this.fileService
              .setFileSettings(result.customUrl, configResult)
              .subscribe(
                (response: any) => {
                    window.location.href = `/${result.customUrl}`;
                },
                (error: any) => {
                  console.error('Error updating file settings:', error);
                }
              );
          }
        });
      }
    });
  }
}

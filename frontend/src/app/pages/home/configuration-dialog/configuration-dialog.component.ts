import { Component, Inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormGroup,
  FormControl,
  Validators,
  ReactiveFormsModule,
  ValidatorFn,
  AbstractControl,
} from '@angular/forms';
import {
  MatDialogRef,
  MAT_DIALOG_DATA,
  MatDialogModule,
} from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon'; // Import MatIconModule
import { FileSettings } from '../../../types/file-settings';

@Component({
  selector: 'app-configuration-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatCheckboxModule,
    MatIconModule, // Add MatIconModule to imports
  ],
  template: `
    <div class="config-dialog">
      <h2 mat-dialog-title>File Configuration</h2>
      <mat-dialog-content>
        <form class="configuration-form" [formGroup]="configForm">
          <mat-form-field
            class="date-input"
            appearance="outline"
            floatLabel="always"
          >
            <mat-label>Expiration Date (optional)</mat-label>
            <input
              matInput
              formControlName="expiresInDate"
              placeholder="DD/MM/YYYY"
              (input)="onDateInput($event)"
            />
            <mat-error
              *ngIf="
                configForm.get('expiresInDate')?.hasError('invalidDateFormat') &&
                configForm.get('expiresInDate')?.touched
              "
            >
              Invalid date format (DD/MM/YYYY)
            </mat-error>
            <mat-error
              *ngIf="
                configForm.get('expiresInDate')?.hasError('invalidDate') &&
                configForm.get('expiresInDate')?.touched
              "
            >
              Invalid date
            </mat-error>
            <mat-error
              *ngIf="
                configForm.get('expiresInDate')?.hasError('dateInPast') &&
                configForm.get('expiresInDate')?.touched
              "
            >
              Date must be in the future
            </mat-error>
          </mat-form-field>

          <mat-form-field appearance="outline">
            <mat-label>Password Protection (optional)</mat-label>
            <input
              matInput
              [type]="showPassword ? 'text' : 'password'"
              formControlName="password"
            />
            <button
              mat-icon-button
              matSuffix
              (click)="togglePasswordVisibility()"
              type="button"
              [attr.aria-label]="
                showPassword ? 'Hide password' : 'Show password'
              "
            >
              <mat-icon class="white-icon">{{
                showPassword ? 'visibility' : 'visibility_off'
              }}</mat-icon>
            </button>
          </mat-form-field>

          <div class="settings-group">
            <mat-checkbox formControlName="deletesAfterDownload">
              Delete after download (optional)
            </mat-checkbox>
            <mat-form-field
              *ngIf="configForm.get('deletesAfterDownload')?.value"
              appearance="outline"
            >
              <mat-label>Number of downloads before deletion</mat-label>
              <input
                matInput
                type="number"
                formControlName="downloadsForDeletion"
              />
              <mat-error
                *ngIf="
                  configForm
                    .get('downloadsForDeletion')
                    ?.hasError('min') &&
                  configForm.get('downloadsForDeletion')?.touched
                "
              >
                Must be at least 1
              </mat-error>
            </mat-form-field>
          </div>

          <div class="settings-group">
            <mat-checkbox formControlName="deletesAfterVizualizations">
              Delete after visualization (optional)
            </mat-checkbox>
            <mat-form-field
              *ngIf="configForm.get('deletesAfterVizualizations')?.value"
              appearance="outline"
            >
              <mat-label>Number of visualizations before deletion</mat-label>
              <input
                matInput
                type="number"
                formControlName="vizualizationsForDeletion"
              />
              <mat-error
                *ngIf="
                  configForm
                    .get('vizualizationsForDeletion')
                    ?.hasError('min') &&
                  configForm.get('vizualizationsForDeletion')?.touched
                "
              >
                Must be at least 1
              </mat-error>
            </mat-form-field>
          </div>
        </form>
      </mat-dialog-content>
      <mat-dialog-actions align="end">
        <button mat-button (click)="onCancel()">Cancel</button>
        <button
          mat-raised-button
          color="primary"
          (click)="onSave()"
          [disabled]="configForm.invalid"
        >
          Save
        </button>
      </mat-dialog-actions>
    </div>
  `,
  styles: [
    `
      .config-dialog {
        padding: 24px;
        background: #1a1a2e;
        color: white;
      }

      .date-input {
        margin-top: 10px;
      }

      h2[mat-dialog-title] {
        color: #b388ff !important;
        margin-bottom: 24px !important;
        font-size: 28px !important;
        font-weight: 600 !important;
        letter-spacing: 0.5px !important;
      }

      .configuration-form {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .settings-group {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
        padding: 1rem;
        border: 1px solid rgba(179, 136, 255, 0.2);
        border-radius: 8px;
      }

      ::ng-deep .mat-mdc-form-field .mdc-notched-outline > * {
        border-color: rgba(179, 136, 255, 0.3) !important;
      }

      ::ng-deep
        .mat-mdc-form-field:not(.mat-form-field-invalid)
        .mdc-notched-outline
        > * {
        border-color: rgba(179, 136, 255, 0.7) !important;
      }

      ::ng-deep .mat-mdc-form-field .mat-mdc-floating-label {
        color: #b388ff !important;
      }

      ::ng-deep .mat-mdc-form-field input {
        color: white !important;
      }

      ::ng-deep .mat-mdc-checkbox .mdc-checkbox__background {
        background-color: transparent !important;
        border-color: rgba(179, 136, 255, 0.7) !important;
      }

      ::ng-deep .mat-mdc-checkbox .mdc-checkbox__checkmark {
        fill: #b388ff !important;
      }

      ::ng-deep .mat-mdc-checkbox .mdc-checkbox__ripple {
        display: none;
      }

      ::ng-deep .mat-mdc-checkbox label {
        color: rgba(255, 255, 255, 0.9) !important;
      }

      ::ng-deep .mat-mdc-dialog-actions {
        padding: 24px 0 0 !important;
        margin: 0 !important;
        gap: 12px !important;
      }

      ::ng-deep .mat-mdc-button {
        color: rgba(255, 255, 255, 0.7) !important;
      }

      ::ng-deep .mat-mdc-button:hover {
        color: white !important;
        background: rgba(255, 255, 255, 0.1) !important;
      }

      ::ng-deep .mat-mdc-raised-button.mat-primary {
        background-color: #b388ff !important;
        color: #1a1a2e !important;
        font-weight: 500 !important;
      }

      ::ng-deep .mat-mdc-raised-button.mat-primary:hover {
        background-color: #9b6bff !important;
      }

      /* Placeholder color */
      ::ng-deep .mat-mdc-form-field input::placeholder {
        color: rgba(255, 255, 255, 0.6) !important;
        caret-color: white !important;
        opacity: 0.7;
      }

      /* Style for the eye icon */
      .white-icon {
        color: white;
      }
    `,
  ],
})
export class ConfigurationDialogComponent implements OnInit {
  configForm!: FormGroup;
  showPassword = false; // Property to control password visibility

  constructor(
    public dialogRef: MatDialogRef<ConfigurationDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { customUrl: string }
  ) {}

  ngOnInit(): void {
    this.configForm = new FormGroup({
      expiresInDate: new FormControl<string | null>(null, [
        this.dateValidator(),
      ]),
      password: new FormControl<string | null>(null),
      deletesAfterDownload: new FormControl<boolean>(false),
      downloadsForDeletion: new FormControl<number | null>(null, [
        Validators.min(1),
      ]),
      deletesAfterVizualizations: new FormControl<boolean>(false),
      vizualizationsForDeletion: new FormControl<number | null>(null, [
        Validators.min(1),
      ]),
    });

    // Listen for changes in checkboxes to apply/clear validators for number fields
    this.configForm
      .get('deletesAfterDownload')
      ?.valueChanges.subscribe((checked) => {
        const downloadsControl = this.configForm.get('downloadsForDeletion');
        if (checked) {
          downloadsControl?.setValidators([Validators.required, Validators.min(1)]);
        } else {
          downloadsControl?.clearValidators();
          downloadsControl?.setValue(null);
        }
        downloadsControl?.updateValueAndValidity();
      });

    this.configForm
      .get('deletesAfterVizualizations')
      ?.valueChanges.subscribe((checked) => {
        const vizualizationsControl = this.configForm.get(
          'vizualizationsForDeletion'
        );
        if (checked) {
          vizualizationsControl?.setValidators([Validators.required, Validators.min(1)]);
        } else {
          vizualizationsControl?.clearValidators();
          vizualizationsControl?.setValue(null);
        }
        vizualizationsControl?.updateValueAndValidity();
      });

    // Prevent initial focus
    setTimeout(() => {
      const activeElement = document.activeElement as HTMLElement;
      if (activeElement) {
        activeElement.blur();
      }
    });
  }

  // Method to toggle password visibility
  togglePasswordVisibility(): void {
    this.showPassword = !this.showPassword;
  }

  // Custom date validator
  dateValidator(): ValidatorFn {
    return (control: AbstractControl): { [key: string]: any } | null => {
      const dateString = control.value;

      if (!dateString) {
        return null; // No validation if empty, as it's optional
      }

      const parts = dateString.split('/');
      if (parts.length !== 3) {
        return { invalidDateFormat: true };
      }

      const day = parseInt(parts[0], 10);
      const month = parseInt(parts[1], 10) - 1; // JavaScript months are 0-based
      const year = parseInt(parts[2], 10);

      const date = new Date(year, month, day);

      // Check if the date components actually form a valid date
      if (
        date.getDate() !== day ||
        date.getMonth() !== month ||
        date.getFullYear() !== year
      ) {
        return { invalidDate: true };
      }

      // Check if the date is in the future
      if (date <= new Date()) {
        return { dateInPast: true };
      }

      return null; // Date is valid
    };
  }

  onDateInput(event: Event): void {
    const input = (event.target as HTMLInputElement).value;
    let cleaned = input.replace(/\D/g, '');

    let formattedDate = '';
    if (cleaned.length > 0) {
      if (cleaned.length <= 2) {
        formattedDate = cleaned;
      } else if (cleaned.length <= 4) {
        formattedDate = `${cleaned.slice(0, 2)}/${cleaned.slice(2)}`;
      } else {
        formattedDate = `${cleaned.slice(0, 2)}/${cleaned.slice(
          2,
          4
        )}/${cleaned.slice(4, 8)}`;
      }
    }

    // Update the form control value, triggering validation
    this.configForm.get('expiresInDate')?.setValue(formattedDate, { emitEvent: false });
    // Manually trigger validation on blur or when the input is complete
    this.configForm.get('expiresInDate')?.markAsTouched();
  }

  onCancel(): void {
    this.dialogRef.close();
  }

  onSave(): void {
    // Mark all controls as touched to display validation errors
    this.configForm.markAllAsTouched();

    if (this.configForm.invalid) {
      return; // Do not proceed if the form is invalid
    }

    const formValue = this.configForm.value;
    const settings: FileSettings = {};

    // Convert expiresInDate string to Date object if valid
    if (formValue.expiresInDate) {
      const parts = formValue.expiresInDate.split('/');
      const day = parseInt(parts[0], 10);
      const month = parseInt(parts[1], 10) - 1;
      const year = parseInt(parts[2], 10);
      settings.expiresIn = new Date(year, month, day);
    }

    if (formValue.password) {
      settings.password = formValue.password;
    }
    if (formValue.deletesAfterDownload) {
      settings.deletesAfterDownload = formValue.deletesAfterDownload;
      settings.downloadsForDeletion = formValue.downloadsForDeletion;
    }
    if (formValue.deletesAfterVizualizations) {
      settings.deletesAfterVizualizations = formValue.deletesAfterVizualizations;
      settings.vizualizationsForDeletion = formValue.vizualizationsForDeletion;
    }

    // Clean up the settings object by removing undefined or null values
    const cleanedSettings = Object.fromEntries(
      Object.entries(settings).filter(([_, value]) => value !== undefined && value !== null)
    );

    this.dialogRef.close(cleanedSettings);
  }
}
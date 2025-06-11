import { Routes } from '@angular/router';
import { Home } from './pages/home/home';
import { FileViewComponent } from './pages/file-view/file-view.component';

export const routes: Routes = [
  { path: '', component: Home },
  { path: ':customUrl', component: FileViewComponent }, // Route for viewing files
];

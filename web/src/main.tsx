import { StrictMode } from 'react';
import ReactDOM from 'react-dom/client';
import { Toaster } from '@/components/ui/sonner';
import { RouterProvider } from '@tanstack/react-router';
import { queryClient, QueryProvider, router } from './lib/root-provider';
import './styles.css';

// Render the app
const rootElement = document.getElementById('app');
if (rootElement && !rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <QueryProvider queryClient={queryClient}>
        <RouterProvider router={router} />
        <Toaster richColors position="top-right" />
      </QueryProvider>
    </StrictMode>,
  );
}

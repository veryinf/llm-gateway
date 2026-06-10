import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
  loader: () => {
    throw redirect({ to: '/dashboard' });
  },
  component: Home,
});

function Home() {
  return null;
}

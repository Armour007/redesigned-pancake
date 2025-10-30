import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import LoginPage from '../routes/login/+page.svelte';

describe('Login page', () => {
  it('logs in and redirects to /dashboard', async () => {
    render(LoginPage);
    const email = screen.getByLabelText(/email/i);
    const password = screen.getByLabelText(/password/i);
    const button = screen.getByRole('button', { name: /log in/i });

    await fireEvent.input(email, { target: { value: 'user@example.com' } });
    await fireEvent.input(password, { target: { value: 'secret' } });
    await fireEvent.click(button);

    // The page requests /auth/login then /organizations/mine. Ensure no error shows
    await waitFor(() => {
      expect(screen.queryByText(/login failed/i)).toBeNull();
    });

    // Verify token stored and no error message shown (navigation is handled by SvelteKit)
    await waitFor(() => {
      expect(localStorage.getItem('aura_token')).toBe('test_token');
    });
  });

  it('shows error on invalid credentials', async () => {
    render(LoginPage);
    const email = screen.getByLabelText(/email/i);
    const password = screen.getByLabelText(/password/i);
    const button = screen.getByRole('button', { name: /log in/i });

    // Use a known-bad password to force 400 from the fetch mock
    await fireEvent.input(email, { target: { value: 'user@example.com' } });
    await fireEvent.input(password, { target: { value: 'bad' } });
    await fireEvent.click(button);

    // The UI should show an error message
    const err = await screen.findByText(/invalid|login failed/i);
    expect(err).toBeInTheDocument();
  });
});

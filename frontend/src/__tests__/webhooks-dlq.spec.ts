import { render, screen, fireEvent } from '@testing-library/svelte';
import WebhooksDLQPage from '../routes/(app)/admin/webhooks/dlq/+page.svelte';

describe('Webhooks DLQ page', () => {
  it('renders table headers', async () => {
    render(WebhooksDLQPage);
    // Wait for list to load
    await screen.findByText(/fetched/i);
    expect(await screen.findByText(/endpoint/i)).toBeInTheDocument();
    expect(await screen.findByText(/event/i)).toBeInTheDocument();
  });

  it('requeues a selected item and shows toast', async () => {
    render(WebhooksDLQPage);
    await screen.findByRole('checkbox', { name: /select x-10/i });
    const firstSelect = await screen.findByRole('checkbox', { name: /select x-10/i });
    await fireEvent.click(firstSelect);
    const btn = screen.getByRole('button', { name: /requeue selected/i });
    await fireEvent.click(btn);
    expect(await screen.findByText(/requeued 1 item/i)).toBeInTheDocument();
  });

  it('deletes a selected item and shows toast', async () => {
    render(WebhooksDLQPage);
    await screen.findByRole('checkbox', { name: /select x-10/i });
    const firstSelect = await screen.findByRole('checkbox', { name: /select x-10/i });
    await fireEvent.click(firstSelect);
    const btn = screen.getByRole('button', { name: /delete selected/i });
    await fireEvent.click(btn);
    expect(await screen.findByText(/deleted 1 item/i)).toBeInTheDocument();
  });
});

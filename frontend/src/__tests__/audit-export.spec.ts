import { render, screen, fireEvent } from '@testing-library/svelte';
import AuditExportPage from '../routes/(app)/organizations/[orgId]/regulator/audit-export/+page.svelte';

describe('Audit Export page', () => {
  it('shows labeled inputs', () => {
    render(AuditExportPage);
    expect(screen.getByLabelText(/cron/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination type/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination$/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/format/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/lookback/i)).toBeInTheDocument();
  });

  it('saves successfully and shows status', async () => {
    // token required for save
    localStorage.setItem('aura_token', 'test_token');
    // ensure org id available even if $app/stores alias isn't applied in browser mode
    localStorage.setItem('aura_org_id', 'org_1');
    render(AuditExportPage);
    const save = screen.getByRole('button', { name: /save/i });
    await fireEvent.click(save);
    await screen.findByText(/saved/i);
  });

  it('shows failure status when API fails', async () => {
    localStorage.setItem('aura_token', 'test_token');
    render(AuditExportPage);
    const dest = screen.getByLabelText(/destination$/i);
    await fireEvent.input(dest, { target: { value: 'fail' } });
    const save = screen.getByRole('button', { name: /save/i });
    await fireEvent.click(save);
    await screen.findByText(/failed to save/i);
  });
});

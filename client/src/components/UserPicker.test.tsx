import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { UserSearchResult } from '../api/user.api';
import UserPicker from './UserPicker';

const { searchUsers } = vi.hoisted(() => ({
  searchUsers: vi.fn(),
}));

vi.mock('../api/user.api', () => ({
  searchUsers,
}));

describe('UserPicker', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('searches for users and clears the query after immediate selection', async () => {
    const onSelect = vi.fn();
    searchUsers.mockResolvedValue([
      {
        id: 'user-2',
        email: 'jamie@example.com',
        username: 'jamie',
        avatarData: null,
      },
    ]);

    render(
      <UserPicker
        scope="tenant"
        placeholder="Add a member by name or email"
        clearAfterSelect
        onSelect={onSelect}
      />,
    );

    fireEvent.change(screen.getByLabelText('Add a member by name or email'), {
      target: { value: 'jamie' },
    });

    await waitFor(() => {
      expect(searchUsers).toHaveBeenCalledWith('jamie', 'tenant', undefined);
    });

    fireEvent.click(await screen.findByRole('button', { name: 'Select jamie' }));

    await waitFor(() => {
      expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'user-2' }));
    });

    expect(screen.getByLabelText('Add a member by name or email')).toHaveValue('');
  });

  it('supports controlled selections and lets the caller clear them', async () => {
    function ControlledHarness() {
      const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>({
        id: 'user-1',
        email: 'alice@example.com',
        username: 'alice',
        avatarData: null,
      });

      return (
        <UserPicker
          scope="tenant"
          placeholder="Search users"
          value={selectedUser}
          onSelect={(nextUser) => setSelectedUser(nextUser)}
        />
      );
    }

    render(<ControlledHarness />);

    expect(screen.getByText('alice')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Change' }));

    await waitFor(() => {
      expect(screen.getByLabelText('Search users')).toBeInTheDocument();
    });
  });
});

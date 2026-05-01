package checkouts

import "database/sql"

type checkoutScanner interface {
	Scan(dest ...any) error
}

func scanCheckout(row checkoutScanner) (checkoutEntry, error) {
	var (
		item              checkoutEntry
		secretID          sql.NullString
		connectionID      sql.NullString
		approverID        sql.NullString
		reason            sql.NullString
		expiresAt         sql.NullTime
		requesterUsername sql.NullString
		approverEmail     sql.NullString
		approverUsername  sql.NullString
	)
	if err := row.Scan(
		&item.ID,
		&secretID,
		&connectionID,
		&item.RequesterID,
		&approverID,
		&item.Status,
		&item.DurationMinutes,
		&reason,
		&expiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Requester.Email,
		&requesterUsername,
		&approverEmail,
		&approverUsername,
	); err != nil {
		return checkoutEntry{}, err
	}

	if secretID.Valid {
		item.SecretID = stringPtr(secretID.String)
	}
	if connectionID.Valid {
		item.ConnectionID = stringPtr(connectionID.String)
	}
	if approverID.Valid {
		item.ApproverID = stringPtr(approverID.String)
	}
	if reason.Valid {
		item.Reason = stringPtr(reason.String)
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		item.ExpiresAt = &value
	}
	if requesterUsername.Valid {
		item.Requester.Username = &requesterUsername.String
	}
	if approverEmail.Valid {
		item.Approver = &userSummary{Email: approverEmail.String}
		if approverUsername.Valid {
			item.Approver.Username = &approverUsername.String
		}
	}
	return item, nil
}

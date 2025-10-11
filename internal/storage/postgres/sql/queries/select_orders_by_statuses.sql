SELECT id, uid FROM orders WHERE accrual_status IN ($1, $2)

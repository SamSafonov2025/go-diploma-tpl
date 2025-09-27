SELECT id, uid, accrual, accrual_status FROM orders WHERE id=$1 LIMIT 1

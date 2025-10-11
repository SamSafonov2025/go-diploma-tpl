INSERT INTO balances(current_balance, uid) VALUES($1, $2) ON CONFLICT(uid) DO UPDATE SET current_balance=(balances.current_balance+$1) WHERE balances.uid=$2

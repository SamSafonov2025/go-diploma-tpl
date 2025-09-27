SELECT uid, current_balance, withdrawn FROM balances WHERE uid=$1 LIMIT 1

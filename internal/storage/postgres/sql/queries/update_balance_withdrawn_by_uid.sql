UPDATE balances SET current_balance=current_balance-$1, withdrawn=withdrawn+$1 WHERE uid=$2

package report_balance

func accountKeyDailySync() map[string]string {
	return map[string]string{
		"locking account_key":     "lock table account_key_daily_balances in ACCESS exclusive mode",
		"locking journal_entries": "lock table journal_entries in ACCESS exclusive mode",
		// upsert account key daily balance
		"upsert account key daily balance": `
				with entries as (
					select
						date(je.entry_time AT TIME ZONE 'Asia/Jakarta')::timestamp AT TIME ZONE 'Asia/Jakarta' + INTERVAL '7 hours' as day,
						a.account_key,
						je.team_id,
						case 
							when je.rollback = true then je.credit * -1
							else je.debit
						end as debit,
						case 
							when je.rollback = true then je.debit * -1
							else je.credit
						end as credit,
					
						
						case 
							when a.balance_type = 'd' then je.debit - je.credit
							when a.balance_type = 'c' then je.credit - je.debit
						end as balance
						
					--	*
					from 
						journal_entries je
					join accounts a on a.id = je.account_id
				),

				summary as (
					select 
						en.day, 
						en.account_key,
						en.team_id,
						sum(en.debit) as debit,
						sum(en.credit) as credit,
						sum(en.balance) as balance
					
					from entries en
					group by en.day, en.account_key, en.team_id
				),

				statdata as (
					select 
						su.day,
						su.account_key,
						su.team_id,
						su.debit,
						su.credit,
						sum(su.balance) over (
							partition by su.account_key, su.team_id
							order by su.day asc
						) as balance
					
					from summary su
				)

				insert into account_key_daily_balances (day, account_key, journal_team_id, debit, credit, balance)
				select day, account_key, team_id, debit, credit, balance from statdata
				on conflict (day, account_key, journal_team_id)
				do update
				set debit=excluded.debit, 
					credit=excluded.credit, 
					balance=excluded.balance
			`,
		// updating start balance
		"updating start balance": `
				with stb as (
					select 
						akdb.id,
						akdb.day,
						akdb.journal_team_id,
						akdb.account_key,
						akdb.balance,
						lag(akdb.balance) over (
							partition by akdb.account_key, akdb.journal_team_id
							order by akdb.day
						) as st_balance
					from account_key_daily_balances akdb 
				)

				update account_key_daily_balances as u 
					set start_balance=datalb.st_balance
				from (
					select * from stb
				) as datalb
				where u.id = datalb.id
			`,
	}
}

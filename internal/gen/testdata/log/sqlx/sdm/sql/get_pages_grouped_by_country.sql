select sum(v.v_pages) as v_pages_sum, c.code2 as c_code2, c.code3 as c_code3, c.name as c_name
from visitors v left outer join countries c on v.c_code2=c.code2
where (v.v_pages > v.v_sessions)
GROUP BY v.c_code2
order by v_pages_sum DESC

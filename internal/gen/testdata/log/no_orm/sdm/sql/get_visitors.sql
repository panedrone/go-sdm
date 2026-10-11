select v.*, c.code3 as c_code3, c.name as c_name
    from visitors v left outer join countries c on v.c_code2=c.code2
    order by v_last_datetime DESC
    limit ?, ?



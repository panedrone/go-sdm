select r.*, c.code3 as c_code3, c.name as c_name
from REQUEST_LOG r left outer join countries c on r.c_code2=c.code2
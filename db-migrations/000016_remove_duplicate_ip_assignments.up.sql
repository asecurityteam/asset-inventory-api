-- select records we will remove into the new backup table in case we need to recover them
select aws_private_ip_assignment.id,
       not_before,
       not_after,
       private_ip,
       aws_resource_id
into backup_aws_private_ip_assignment
from aws_private_ip_assignment
inner join aws_resource ON aws_private_ip_assignment.aws_resource_id = aws_resource.id
where aws_resource.arn_id = 'i-090efae3665ce3901';

-- delete duplicate records for this resource id
delete
from aws_private_ip_assignment
inner join aws_resource ON aws_private_ip_assignment.aws_resource_id = aws_resource.id
where aws_resource.arn_id = 'i-090efae3665ce3901';

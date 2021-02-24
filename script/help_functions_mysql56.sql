SELECT distinct hk.name as keyword_name
FROM help_keyword AS hk
LEFT JOIN help_relation AS hr ON hr.help_keyword_id = hk.help_keyword_id
LEFT JOIN help_topic AS ht ON ht.help_topic_id = hr.help_topic_id
LEFT JOIN help_category AS hc ON hc.help_category_id = ht.help_category_id
where hc.parent_category_id IN (4, 8, 21)
order by keyword_name

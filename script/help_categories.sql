SELECT hc1.help_category_id,
       hc1.name,
       hc2.help_category_id,
       hc2.name
FROM help_category AS hc1
JOIN help_category AS hc2 ON hc1.help_category_id = hc2.parent_category_id
ORDER BY hc1.help_category_id

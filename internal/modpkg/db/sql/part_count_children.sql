
SELECT
  count(*)
FROM
  part
WHERE
  package_id IS NOT NULL
AND
  rowid IN (
    SELECT DISTINCT
      descendant_id
    FROM
      part_closure
    WHERE
      ancestor_id = $part_id
    )
;
 

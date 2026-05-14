-- name: CreateUniversalConversion :exec
INSERT INTO conversions (ingredient_id, from_unit, to_unit, ratio)
VALUES (
	$1,
	$2,
	$3,
	$4
);

-- name: CreateRetailConversion :exec
INSERT INTO retail_conversions (universal_unit, retail_unit, ratio)
VALUES (
	$1,
	$2,
	$3
);

-- name: GetConversionsByID :many
SELECT * FROM conversions
WHERE ingredient_id = $1;

-- name: CreateUnit :exec
INSERT INTO units (name, abbreviation)
VALUES (
	$1,
	$2
);

-- name: CreateUniversalUnit :exec
INSERT INTO universal_units (name)
VALUES(
	$1
);

-- name: CreateRetailUnit :exec
INSERT INTO retail_units (name)
VALUES (
	$1
);

-- name: CreateIngredientRetailUnit :exec
INSERT INTO ingredient_retail_units (ingredient_id, retail_unit)
VALUES (
	$1,
	$2
);

-- name: GetRetailConversion :many
SELECT * FROM retail_conversions
WHERE retail_unit IN (
	SELECT retail_unit FROM ingredient_retail_units
	WHERE ingredient_id = $1)
AND universal_unit = $2;

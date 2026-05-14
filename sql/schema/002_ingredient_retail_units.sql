-- +goose Up
CREATE TABLE ingredient_retail_units (
	ingredient_id UUID NOT NULL,
	retail_unit TEXT NOT NULL,
	FOREIGN KEY (ingredient_id) REFERENCES ingredients(id),
	FOREIGN KEY (retail_unit) REFERENCES retail_units(name),
	PRIMARY KEY (ingredient_id, retail_unit)
);

-- +goose Down
DROP TABLE ingredient_retail_units;

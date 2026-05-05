-- +goose Up
CREATE TABLE users(
        id UUID PRIMARY KEY,
        name TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        updated_at TIMESTAMP NOT NULL,
        email TEXT NOT NULL UNIQUE,
        hashed_pw TEXT NOT NULL,
	admin BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE recipes(
        id UUID PRIMARY KEY,
        title TEXT NOT NULL,
        author TEXT NOT NULL,
        description TEXT NOT NULL DEFAULT 'Theres nothing here',
	instructions TEXT NOT NULL DEFAULT 'Theres nothing here',
        image_key TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMP NOT NULL,
        updated_at TIMESTAMP NOT NULL,
        user_id UUID NOT NULL,
        CONSTRAINT fk_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE ingredients(
        id UUID PRIMARY KEY,
        name TEXT NOT NULL,
        image_key TEXT NOT NULL DEFAULT '',
        created_at TIMESTAMP NOT NULL,
        updated_at TIMESTAMP NOT NULL
);

CREATE TABLE units (
        name TEXT PRIMARY KEY,
        abbreviation TEXT NOT NULL
);

CREATE TABLE retail_units (
        name TEXT PRIMARY KEY
);

CREATE TABLE conversions (
        ingredient_id UUID NOT NULL,
        from_unit TEXT NOT NULL,
        to_unit TEXT NOT NULL,
        ratio REAL NOT NULL,
        FOREIGN KEY (ingredient_id) REFERENCES ingredients(id),
        FOREIGN KEY (from_unit) REFERENCES units(name),
        FOREIGN KEY (to_unit) REFERENCES retail_units(name),
        PRIMARY KEY (ingredient_id, from_unit, to_unit)
);

CREATE TABLE refresh_tokens(
        id TEXT PRIMARY KEY,
        created_at TIMESTAMP NOT NULL,
        revoked_at TIMESTAMP,
        expires_at TIMESTAMP NOT NULL,
        user_id UUID NOT NULL,
        CONSTRAINT fk_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE TABLE recipe_ingredients (
        recipe_id UUID NOT NULL,
        ingredient_id UUID NOT NULL,
        quantity REAL NOT NULL,
        unit TEXT NOT NULL,
	FOREIGN KEY (unit) REFERENCES units(name),
        FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
        FOREIGN KEY (ingredient_id) REFERENCES ingredients(id),
        PRIMARY KEY (recipe_id, ingredient_id)
);

CREATE TABLE shopping_lists (
        id UUID PRIMARY KEY,
        name TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        updated_at TIMESTAMP NOT NULL,
        user_id UUID NOT NULL,
        CONSTRAINT fk_users
        FOREIGN KEY (user_id)
        REFERENCES users(id)
);

CREATE TABLE shopping_list_recipes (
        shopping_list_id UUID NOT NULL,
        recipe_id UUID NOT NULL,
        quantity INT NOT NULL,
        FOREIGN KEY (shopping_list_id) REFERENCES shopping_lists(id) ON DELETE CASCADE,
        FOREIGN KEY (recipe_id) REFERENCES recipes(id),
        PRIMARY KEY (shopping_list_id, recipe_id)
);

-- +goose Down
DROP TABLE conversions, retail_units, units, shopping_list_recipes, shopping_lists, recipe_ingredients, refresh_tokens, ingredients, recipes, users;

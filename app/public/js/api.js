// Fetch with token
async function authorizedFetch(url, options = {}) {
	const token = localStorage.getItem('token');

	options.headers = {
		...options.headers,
		'Authorization': `Bearer ${token}`
	    };

	try {
		let response = await fetch(url, options)

		if (response.status === 401) {
			const ok = await refreshToken();
			if (ok) {
				options.headers['Authorization'] = `Bearer ${localStorage.getItem('token')}`;
				response = await fetch(url, options);
			} else {
				window.location.href = '/login'
				return
			}
		}

		return response

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Rerresh token
async function refreshToken() {
	try {
		let response = await fetch('/api/tokens/refresh', {
		    method: 'POST',
		    headers: {
			Authorization: `Bearer ${localStorage.getItem('refresh_token')}`,
		    },
		});
		if (!response.ok) return false

		const data = await response.json();
		if (data.token) {
		    localStorage.setItem('token', data.token);
		}
		return true
	} catch {
		return false
	}
}

// Get user shopping lists
export async function getShoppingListsHTML() {
        const id = localStorage.getItem('user_id');

	try {
		let response = await authorizedFetch(`/users/${id}/shoppinglists`, {
			headers: {
				'Accept': 'text/html'
			}
		});

		if (!response.ok) {
			throw new Error("Failed to fetch shopping lists")
		}

		const newHTML = await response.text();
		document.open();
		document.write(newHTML);
		document.close();
		window.history.pushState({}, '', `users/${id}/shoppinglists`);
		return

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Get lists for modal
export async function getShoppingListsJSON() {
	const id = localStorage.getItem('user_id');

	try {
		let response = await authorizedFetch(`/users/${id}/shoppinglists`, {
			headers: {
				'Accept': 'application/json'
			}
		});

		if (!response.ok) {
			throw new Error("Failed to fetch shopping lists")
		}

		return await response.json();

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Get recipe cards
export async function getRecipes() {
    try {
	    let response = await fetch('/api/recipes', {})

	    if (!response.ok) {
		    throw new Error("Failed to fetch recipes")
	    }

	    return await response.json();
    } catch (error) {
	    alert(`Error: ${error.message}`)
    }
}

// Login/Signup calls
export async function login() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;

    try {
        const response = await fetch('/api/sessions', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                email,
                password
            }),
        });
        const data = await response.json();
        if (!response.ok) {
            throw new Error(`Failed to login: ${data.error}`);
        }

	return data;

    } catch (error) {
        alert(`Error: ${error.message}`);
    }
}

export async function signup() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    const name = document.getElementById('username').value;

    try {
        const response = await fetch('/api/users', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                email,
                password,
                name
            }),
        });
        if (!response.ok) {
            const data = await response.json();
            throw new Error(`Failed to create user: ${data.error}`);
        }
	alert(`Signup successful...redirecting to login page`)
        window.location.href = '/login';
    } catch (error) {
        alert(`Error: ${error.message}`);
    }
}

// Get user profile page
export async function getUserPage(userID) {
	try {
		let response = await authorizedFetch(`/users/${userID}`, {
			method: 'GET',
			headers: {
				'Accept': 'text/html',
			},
		});
		if (!response.ok) {
			const data = await response.json();
			throw new Error(`Couldn't fetch user: ${data.error}`)
		}

		const newHTML = await response.text();
		document.open();
		document.write(newHTML);
		document.close();
		window.history.pushState({}, '', `/users/${userID}`);
		return

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Post new recipe
export async function postRecipe(form) {
	try {
            const recipeData = {
                title: document.getElementById('recipe-title').value,
                user_id: localStorage.getItem('user_id'),
                ingredients: [],
                description: document.getElementById('author-description').value,
		instructions: document.getElementById('recipe-instructions').value,
            };

            const rows = document.querySelectorAll('.ingredient-row')

            rows.forEach(row => {
                const ingredient = {
                    id: row.querySelector('input[name="ingredient_id"]').value,
                    quantity: parseFloat(row.querySelector('input[name="quantity"]').value),
                    unit: row.querySelector('select[name="unit"]').value
                };
                recipeData.ingredients.push(ingredient);
            });

	    const formData = new FormData();
            formData.append("payload", JSON.stringify(recipeData))
            formData.append("image", form.recipe_pic.files[0])

	    let response = await authorizedFetch('/api/recipes', {
		    method: 'POST',
		    body: formData,
	    });

	   if (response.ok) {
		   alert("Success!");
		   const posted = await response.json();
		   const link = posted.id;
		   window.location.href = `/recipes/${link}`;
	   }
	} catch (error) {
		alert(`Error: ${error.message}`)
	}
}

// Create new list
export async function postShoppingList() {
	try {
		const name = document.getElementById('shopping-list-title').value;

		let response = await authorizedFetch('/api/shoppinglists', {
			method: 'POST',
			body: JSON.stringify({ name: name }),
		});

		if (response.ok) {
			alert("Success!");
			window.location.href = '/';
			return
		}
	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Add recipe to list
export async function postAddtoList(shoppingListID, recipeID) {
	try {
		
		const response = await authorizedFetch(`/api/shoppinglists/${shoppingListID}`, {
			method: 'POST',
			body: JSON.stringify({ shopping_list_id: shoppingListID, recipe_id: recipeID, quantity: 1 })
		});

		if (response.ok) {
			alert("Success!");
		}

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

// Fetch shopping list page
export async function getUserShoppingList(id) {
	try {
		const response = await authorizedFetch(`/shoppinglists/${id}`);

		const newHTML = await response.text();
		document.open();
		document.write(newHTML);
		document.close();
		window.history.pushState({}, '', `/shoppinglists/${id}`);
		return

	} catch (error) {
		alert(`Error: ${error.message}`);
	}
}

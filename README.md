# Forum

A full-stack discussion forum built with Go. Users can sign up, create posts, comment, like/dislike content, and sort posts by various criteria. The application features session-based authentication, a clean responsive UI, and vanilla JavaScript for interactive elements.

<img src="https://github.com/user-attachments/assets/374695e4-82bd-4169-9738-6e84a18864e0" alt="Homepage Screenshot" style="max-width:100%;"/>

<img src="https://github.com/user-attachments/assets/4e0a2087-a534-4c9d-8059-2492ec14a370" alt="SignUp Screenshot" style="max-width:100%;"/>

<img src="https://github.com/user-attachments/assets/417a394a-9587-4f5c-a08a-4452d219b9dd" alt="Post Screenshot" style="max-width:100%;"/>

## Features

### User Authentication
- **Sign Up / Log In** – Register with email, username, and password.  
- **Secure Sessions** – Cookie-based authentication with expiration and bcrypt password hashing.  
- **Profile Settings** – Update username, email, or password.

### Posts
- **Create Posts** – Add a title, content, and assign categories (e.g., `#tigrBrat228` in the screenshot).  
- **Sorting** – Choose from:
  - Newest First  
  - Oldest First  
  - Recently Updated  
  - Most Liked  
  - Most Disliked  
- **Edit & Delete** – Post owners can edit or delete their own posts.

### Comments
- **Add Comments** – Leave comments on any post.  
- **Edit & Delete** – Comment owners can modify or remove their comments.

### Likes & Dislikes
- **Vote on Posts & Comments** – Click the like/dislike icons to express your opinion.  
- **Real-time UI Feedback** – Vanilla JavaScript ensures mutual exclusion (you cannot like and dislike the same item) and highlights the active sort option without page reload.

### Categories
- Posts can belong to one or more categories (visible as hashtags).  
- Filter posts by category (if implemented).

### Responsive Design
- The interface adapts to mobile and desktop screens using custom CSS.

## Tech Stack

- **Backend:** Go (standard library, custom routing, middleware)  
- **Frontend:** HTML templates, CSS, **vanilla JavaScript** (no frameworks)  
- **Database:** SQLite (with `github.com/mattn/go-sqlite3`)  
- **Authentication:** Session-based cookies, bcrypt (`golang.org/x/crypto`)  
- **Additional Packages:**  
  - `github.com/google/uuid` – for generating unique session IDs

## Installation

### Prerequisites
- Go 1.20 or higher
- Git

### Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/TerZoro/forum.git
   cd forum
   ```

2. **Install dependencies**  
   The project uses Go modules. Download all required packages:
   ```bash
   go mod download
   ```

3. **Set up the database**  
   A pre-configured SQLite file `forumdb.db` is included. To start fresh, delete it and the application will recreate the schema on first run.

4. **Run the application**
   ```bash
   go run cmd/web/main.go
   ```
   You should see output similar to:
   ```
   2026/03/10 00:43:34 Using database file: /path/to/forum/forumdb.db
   2026/03/10 00:43:34 Database connection successful
   2026/03/10 00:43:34 Repository initialized successfully
   2026/03/10 00:43:34 Service layer initialized
   2026/03/10 00:43:34 Templates loaded successfully
   2026/03/10 00:43:34 All routes registered successfully
   2026/03/10 00:43:34 Starting server on port :8080...
   ```

5. **Open in your browser**  
   Visit `http://localhost:8080` to start using the forum.

## Running with Docker

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) installed and running

### Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/TerZoro/forum.git
   cd forum
   ```

2. **Make the scripts executable**
   ```bash
   chmod +x docker_builder.sh docker_runner.sh
   ```

3. **Build the Docker image**
   ```bash
   ./docker_builder.sh
   ```
   This builds the image tagged as `forum:v1`.

4. **Run the container**
   ```bash
   ./docker_runner.sh
   ```
   - If `forumdb.db` doesn't exist yet, it will be created automatically.
   - The database file is mounted as a volume so your data persists between runs.
   - The app will be available at `http://localhost:8080`.

5. **Stop the container**
   Press `Ctrl+C` in the terminal. The container is removed automatically (`--rm` flag).

> **Note:** Run `docker_builder.sh` once (or whenever you make code changes), then use `docker_runner.sh` each time you want to start the app.

---

## Usage

1. **Sign Up** – Create a new account using the sign-up page.

2. **Log In** – Use your credentials to access the forum.

3. **Create a Post** – Click the "Create Post" button, fill in the title, content, and categories.

4. **Interact** – Browse posts, leave comments, and like/dislike content. The JavaScript ensures that like/dislike states update instantly and remain mutually exclusive.

5. **Manage Your Content** – Edit or delete your own posts and comments.

## JavaScript Highlights

The frontend uses lightweight vanilla JavaScript to enhance user experience without heavy frameworks. Key features:

- **Sorting UI Highlight** – The currently active sort option (e.g., "Newest First") is visually highlighted.
- **Like/Dislike Mutual Exclusion** – Clicking a like button automatically removes any existing dislike on the same item (and vice versa), providing instant visual feedback.

You can find the JavaScript code in the project's static files (e.g., `static/js/main.js`).

## Project Structure

```
forum/
├── cmd/
│   └── web/                # Main application entry point (main.go)
├── internal/                # Private application code
│   ├── handlers/            # HTTP request handlers
│   ├── models/              # Data models and database logic
│   ├── middleware/          # Custom middleware (auth, logging)
│   └── ...                  # Other internal packages
├── static/                  # CSS, JavaScript, images
├── templates/               # HTML templates
├── forumdb.db               # SQLite database file (optional)
├── Dockerfile               # Multi-stage Docker build
├── docker_builder.sh        # Script to build the Docker image
├── docker_runner.sh         # Script to run the container with volume mount
├── go.mod                   # Module definition
├── go.sum                   # Module checksums
└── README.md                # This file
```

## Contributing

This is a personal learning project, but contributions are welcome! Feel free to open issues or pull requests for bug fixes, improvements, or new features.

## License

This project is licensed under the MIT License – see the [LICENSE](LICENSE) file for details.

# Development

## Initial setup

1. Install prerequisites

    On mac:

    ```sh
    brew install --cask docker-desktop
    brew install go
    brew install npm
    brew install direnv
    ```

    Add this line to your `~/.zshrc` file:

    ```sh
    eval "$(direnv hook zsh)"
    ```


2. Install Go dependencies

    ```sh
    go mod tidy
    ```

3. Install Javascript dependencies

    ```sh
    cd app
    npm install
    cd ..
    ```

4. Start application dependencies

    ```sh
    docker compose up --remove-orphans
    ```

5. Setup configuration

    ```sh
    cp .env.example .env
    ```

6. Add Hatchett client token to `.env`

    1. Point your browser to the [Hatchett admin UI](http://localhost:8888/auth/login) and sign in with `admin@example.com` / `Admin123!!`.
    2. In Settings / API Tokens, choose Create API Token.
    3. Add the token to your `.env` file.
    4. Run
        ```sh
        direnv allow
        ```

7. Create database structure

    ```sh
    go run cmd/bbl/main.go migrate up
    ```

8. Start the application in development mode

    ```sh
    make live
    ```

    In development mode the application will reload itself after a `.go` source file, a `.templ` template, a file in the assets directory, a `.po` translation file or a profile `.json` file has changed.

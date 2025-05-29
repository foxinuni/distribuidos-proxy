# Central y Facultades

Este repositorio contiene el código del proxy utilizado como Load Balancer y Health Checker. Es sencillo de ejecutar y compatible con Windows.

## Dependencias

- **sqlc:** Generador de código a partir de esquemas SQL.
- **migrate:** Gestor de migraciones para bases de datos.
- **go-task:** Herramienta de automatización de tareas.
- **go:** Lenguaje de programación principal.

> **Nota:** No es necesario instalar estas dependencias si ejecutas el proyecto con Docker.  
> Docker instalará automáticamente todo lo necesario, sin importar tu sistema operativo.

## Compilación

Para compilar el programa, asegúrate de tener Docker instalado.  
Puedes descargarlo desde: [https://www.docker.com](https://www.docker.com).

1. Abre una terminal en la carpeta del proyecto:
    ```sh
    cd distribuidos-central
    ```
2. Construye la imagen Docker (esto puede tardar unos minutos):
    ```sh
    # "dist-proxy" es el nombre de la imagen a crear
    docker build -t dist-proxy .
    ```

3. Ejecuta el proxy:
    ```sh
    docker run --rm \
      --network host \
      dist-proxy proxy -port 4444 -workers 10
    ```

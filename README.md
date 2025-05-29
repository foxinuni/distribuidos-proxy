# Central y Facultades

El siguiente código compila los programas de la central y las facultades.  
En este caso, la manera más fácil de compilar es a través de Docker.

### Dependencias

- **libsodium, libczmq, libzmq:** Librerías dependencias para ZeroMQ en Golang.
- **sqlc:** Generador de código a partir de esquemas SQL.
- **migrate:** Gestionador de migraciones para base de datos.
- **go-task:** Herramienta de compilación.
- **go:** Lenguaje de programación utilizado.

> **Importante:** No es necesario instalar ninguna de estas dependencias al ejecutar desde Docker.  
> Docker instala todas las dependencias necesarias para correr el proyecto, independientemente del sistema operativo.

### Compilación

Para compilar el programa, como se mencionó anteriormente, se necesita tener Docker instalado en la máquina.  
Docker se puede instalar desde su página oficial: [https://www.docker.com](https://www.docker.com).

1. Abre una terminal dentro de la carpeta del proyecto:
    ```sh
    cd distribuidos-central
    ```
2. Construye la imagen que contiene los binarios (esto puede tardar unos minutos):
    ```sh
    # "dist-tools" es el nombre de la imagen a crear
    docker build -t dist-tools .
    ```

A partir de este momento, tienes a tu disposición una serie de binarios y utilidades para desplegar los servicios.  
Cada uno de estos binarios tiene parámetros configurables, garantizando portabilidad. Los binarios son los siguientes:

#### 1. migrate

Permite migrar los esquemas que inicializan la base de datos.

```sh
# La URL de la base de datos debe seguir el estándar de Postgres.
# Ejemplo: postgres://user:password@127.0.0.1/db?sslmode=disable
docker run --rm \
  --network host \
  -e DATABASE_URL=${DATABASE_URL} \
  dist-tools task migrate
```

#### 2. populate

Llena la base de datos con salones y laboratorios.

```sh
# Se puede cambiar el número de salones de cada tipo cambiando los parámetros.
docker run --rm \
  --network host \
  dist-tools populate -classrooms 350 -laboratories 100 -database ${DATABASE_URL}
```

#### 3. central

Inicia el servidor central.

```sh
# Si no se especifica el número de trabajadores se utiliza el número de cores de la máquina.
docker run --rm \
  --network host \
  dist-tools central -port 5555 -workers 20 -database ${DATABASE_URL}
```

#### 4. faculty

Inicia la prueba de facultades.

```sh
# Es importante dar la dirección del servidor central en el formato: tcp://[ip]:[puerto]
docker run --rm \
  --network host \
  -v ./logs/:/app/logs \
  dist-tools faculty -faculties 10 -address tcp://127.0.0.1:5555
```
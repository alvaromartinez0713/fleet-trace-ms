# Arquitectura del proyecto

Organizamos el código con Vertical Slice, aplicando los principios de Clean Architecture. Cada servicio es una unidad propia, y dentro de él cada feature es un slice. Un slice contiene todo lo que necesita esa feature: su caso de uso, la entrada que lo invoca (HTTP, consumidor de mensajes, etc.) y las interfaces que requiere. No usamos carpetas técnicas globales por capa (domain, application, infrastructure) para repetir la misma estructura en cada feature.

## Estructura de un servicio

```
services/<servicio>/
├── cmd/                      # Punto de entrada: carga config, maneja señales y llama a app.Run
└── internal/
    ├── app/                  # Composition root: construye externos, servicios, handlers, router y server
    ├── domain/               # Entidades y reglas de negocio compartidas por el servicio
    ├── <feature>/            # Slice de una feature (ej. reportlocation)
    │   ├── service.go        # Caso de uso: flujo de la feature
    │   ├── <interfaz>.go     # Interfaces que la feature necesita (ej. publisher.go)
    │   ├── handler.go        # Entrada HTTP o de mensajes de la feature
    │   └── dto.go            # Request y response HTTP (o de mensajes) de la feature
    └── external/             # Implementaciones de sistemas externos que usa el servicio
        ├── kafka/
        └── ...               # Otros servicios externos (HTTP, bases de datos, etc.)
```

Código compartido por todos los servicios vive en `internal/platform/` en la raíz del módulo (config, logging, httpserver, cliente Kafka genérico).

## Reglas de dependencia

- La dependencia apunta hacia adentro. `domain/` no importa features, `external/` ni `platform/` específicos de infraestructura.
- Cada feature importa `domain/` y define en su propio paquete las interfaces que necesita.
- `external/` implementa esas interfaces. Ninguna implementación concreta se importa directamente desde una feature.
- Una feature no importa a otra feature. Si dos features necesitan lo mismo, ese código va a `domain/` o a `external/`, según corresponda.
- `platform/` contiene solo código genérico y reutilizable, sin reglas de negocio.
- `internal/app/` es el único lugar donde se conectan las implementaciones con las interfaces. `cmd/` solo carga la configuración y arranca la aplicación, sin cablear dependencias.
- Frameworks de transporte (ej. Gin) solo aparecen en `platform/httpserver/` y en los `handler.go` de cada feature. Nunca en `service.go`, `domain/` ni `external/`.

## Al agregar código

- Crear nuevas funcionalidades como slices nuevos dentro del servicio, reutilizando `domain/` y `external/`. No duplicar entidades ni clientes.
- Antes de mover código a `domain/`, `external/` o `platform/`, confirmar que realmente es compartido.

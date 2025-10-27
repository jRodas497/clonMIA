# Manual Técnico – GoDisk 2.0

### Universidad de San Carlos de Guatemala
### Facultad de Ingeniería – Ingeniería en Ciencias y Sistemas
### Curso: Manejo e Implementación de Archivos
### Proyecto 2 – Segundo Semestre 2025

## 1. Resumen

Este proyecto consiste en la evolución del sistema de archivos simulado del Proyecto 1, con el objetivo de mejorar su accesibilidad y usabilidad mediante una interfaz gráfica web (GUI).
La solución integra frontend (React/Vite) y backend (Go) desplegados en AWS, con las siguientes características:

- Visualización jerárquica de discos, particiones y archivos.

- Soporte completo para EXT2 y EXT3.

- Implementación de journaling para operaciones registradas.

- Ejecución de comandos desde una terminal web.

- Infraestructura distribuida entre S3 (frontend) y EC2 (backend).

## 2. Competencias Desarrolladas

- Implementación de sistemas de archivos EXT2/EXT3 mediante API y GUI web.

- Integración de soluciones de almacenamiento local y en la nube (AWS).

- Análisis de arquitecturas de virtualización de almacenamiento.

## 3. Objetivos del Aprendizaje
### 3.1 Objetivo General

Diseñar y desarrollar una aplicación web que permita la visualización y gestión de un sistema de archivos simulado (EXT2/EXT3) con backend en Go y frontend web, desplegado en AWS.

### 3.2 Objetivos Específicos

- Desarrollar un sistema funcional basado en la nube (EC2 + S3).

- Diseñar interfaces gráficas intuitivas para el manejo visual del sistema de archivos.

- Documentar técnicamente la arquitectura, estructuras y comandos.

## 4. Arquitectura General del Sistema
### 4.1 Componentes Principales 

#### Frontend

- Framework: React + Vite

- Funciones principales:

    - Ejecución de comandos mediante una terminal embebida.

    - Autenticación (login/logout).

    - Visualización de discos, particiones y archivos.

- Despliegue: AWS S3 (sitio web estático).

#### Backend

- Lenguaje: Go (Golang)

- Tipo: API RESTful.

- Funciones principales:

    - Ejecución lógica de comandos (mkdisk, fdisk, mount, mkfs, etc.).

    - Simulación de estructuras EXT2/EXT3 dentro de archivos .mia.

    - Respuesta JSON hacia el frontend.

- Despliegue: AWS EC2 con Ubuntu/Linux.

## 5. Flujo de la Aplicación
### Página Principal (Home)

Contiene la terminal de comandos, donde se crean discos, particiones y archivos mediante comandos.
Incluye un botón “Iniciar Sesión”.

### Login

Permite iniciar sesión gráficamente con usuario, contraseña y partición.
Los comandos que requieren sesión activa son:

```
MKGRP, RMGRP, MKUSR, RMUSR, CHOWN, CHMOD, entre otros.
```

### Visualizador del Sistema de Archivos

Una vez logueado, el usuario podrá explorar los discos y particiones:

**1. Selección de Disco: muestra nombre, capacidad, fit y particiones montadas.**

**2. Selección de Partición: muestra tamaño, tipo de fit y estado (activa/inactiva).**

**3. Explorador de Archivos: muestra carpetas, archivos y permisos.**

**4. Visualizador de Texto: permite abrir archivos de texto en modo lectura.**

### Cierre de Sesión

Debe existir un botón “Cerrar Sesión”, que invalide la sesión activa y limpie los comandos dependientes del usuario.

## 6. Sistema de Archivos EXT3
### 6.1 Estructuras Internas

- Superbloque: metadatos del sistema.

- Inodos: representan archivos y carpetas.

- Bloques: contienen datos (contenido, carpetas o archivos).

- Bitmap: control de uso de bloques e inodos.

- Journal: bitácora de operaciones.

La fórmula para calcular estructuras es:
```
tamaño_particion = sizeOf(superblock) + n * sizeOf(Journaling) + n + 3n + n*sizeOf(inodos) + 3n*sizeOf(block)
```

## 7. Journal (Bitácora)

Registra cada operación realizada (comando, usuario, fecha, hora, resultado).
Debe visualizarse dentro de la interfaz web.
Cada registro posee:

- Operación ejecutada

- Ruta afectada

- Contenido

- Fecha y hora

## 8. Comandos Implementados
### Administración de Particiones


| **Comando**            | **Descripción**                                                                                  |
| ---------------------- | ------------------------------------------------------------------------------------------------ |
| `FDISK (ADD / DELETE)` | Permite crear, eliminar o redimensionar particiones según el tamaño, unidad y tipo especificado. |
| `UNMOUNT`              | Desmonta una partición utilizando su identificador único (`id`).                                 |
| `MKFS (FS)`            | Realiza el formateo completo de una partición en `EXT2` o `EXT3`.                                |

### Manejo de Archivos y carpetas
| **Comando** | **Descripción**                                                                       |
| ----------- | ------------------------------------------------------------------------------------- |
| `REMOVE`    | Elimina archivos o carpetas recursivamente si el usuario tiene permisos de escritura. |
| `EDIT`      | Modifica el contenido de un archivo existente reemplazando su información.            |
| `RENAME`    | Cambia el nombre de un archivo o carpeta, validando duplicados y permisos.            |
| `COPY`      | Copia archivos o carpetas (y su contenido) hacia otro destino.                        |
| `MOVE`      | Mueve archivos o carpetas entre directorios, actualizando las referencias internas.   |
| `FIND`      | Busca archivos o carpetas por nombre, admitiendo comodines (`*`, `?`).                |

### Manejo de Usuarios y Permisos
| **Comando** | **Descripción**                                                                            |
| ----------- | ------------------------------------------------------------------------------------------ |
| `CHOWN`     | Cambia el propietario de archivos o carpetas; sólo `root` o el propietario pueden hacerlo. |
| `CHMOD`     | Modifica los permisos (U/G/O) de archivos o carpetas, de forma individual o recursiva.     |

## 9. Recuperación del Sistema
| **Comando**  | **Función**         | **Descripción**                                                                                    |
| ------------ | ------------------- | -------------------------------------------------------------------------------------------------- |
| `RECOVERY`   | Restauración        | Recupera el sistema de archivos EXT3 a un estado consistente mediante el *journal* y *superblock*. |
| `LOSS`       | Simulación de fallo | Simula pérdida de datos limpiando bitmaps e inodos para probar recuperación.                       |
| `JOURNALING` | Bitácora            | Muestra todas las operaciones registradas con su fecha, hora, usuario y ruta.                      |

## 10. Despliegue en AWS
| **Componente** | **Servicio AWS**       | **Descripción**                                           |
| -------------- | ---------------------- | --------------------------------------------------------- |
| Frontend       | S3 (Static Website)    | Aloja la interfaz web desarrollada en React o Vite.       |
| Backend        | EC2 (Linux Ubuntu)     | API REST desarrollada en Go que procesa comandos.         |
| Comunicación   | HTTP (CORS habilitado) | Intercambio de datos entre frontend (S3) y backend (EC2). |

## 11. Documentación del Proyecto
| **Elemento**                 | **Contenido**                                                      |
| ---------------------------- | ------------------------------------------------------------------ |
| **Arquitectura del sistema** | Diagramas de conexión entre frontend, backend y AWS.               |
| **Estructuras internas**     | MBR, EBR, inodos, bloques, bitmaps, journal y superbloque.         |
| **Comandos implementados**   | Descripción, parámetros, ejemplos y efectos sobre las estructuras. |
| **Despliegue en AWS**        | Pasos para subir frontend a S3 y backend a EC2.                    |

## 12. Metodología SCRUM
| **Sprint** | **Objetivo principal**  | **Entregables esperados**                            |
| ---------- | ----------------------- | ---------------------------------------------------- |
| **1**      | Revisión del Proyecto 1 | `MKDISK`, `FDISK`, `MOUNT`, `MKFS`.                  |
| **2**      | Implementación de EXT3  | Estructuras EXT3, bitmaps, inodos y bloques.         |
| **3**      | Desarrollo del Frontend | Terminal web, carga de scripts, conexión API REST.   |
| **4**      | Pruebas y despliegue    | Deploy en AWS, documentación técnica y manual final. |

## 13. Requisitos Mínimos
| **Requisito**      | **Condición**                        |
| ------------------ | ------------------------------------ |
| Lenguaje Backend   | Go (Golang).                         |
| Framework Frontend | React, Angular o Vue.                |
| Proveedor Cloud    | AWS (S3 + EC2).                      |
| Ejecución          | Script `.smia` completo y funcional. |
| Documentación      | Manual técnico y manual de usuario.  |

## 14. Entregables Finales
| **Entregable**         | **Descripción**                                                   |
| ---------------------- | ----------------------------------------------------------------- |
| Sitio Web Funcional    | Interfaz desplegada en AWS, con conexión activa al backend.       |
| Comandos Implementados | Comandos de administración de discos, usuarios y archivos.        |
| Reportes Visuales      | MBR, inodos, bloques y árbol de directorios (usando Graphviz).    |
| Documentación Técnica  | Manual con arquitectura, estructuras y comandos.                  |
| Manual de Usuario      | Instrucciones, capturas y resolución de errores.                  |
| Repositorio GitHub     | Proyecto completo en repositorio privado con acceso a auxiliares. |

## 15. Rúbrica de Evaluación (Resumen)
| **Área**                            | **Ponderación (pts)** |
| ----------------------------------- | --------------------- |
| Aplicación Web (Frontend + Backend) | 40                    |
| Nuevos Comandos                     | 30                    |
| Journaling y Recuperación           | 15                    |
| Documentación y Conocimientos       | 10                    |
| Comandos Previos                    | 5                     |
| **Total**                           | **100 pts**           |

## 16. Valores y Ética Profesional
| **Principio**           | **Descripción**                                                               |
| ----------------------- | ----------------------------------------------------------------------------- |
| Originalidad            | Cada estudiante debe desarrollar su propio código y documentación.            |
| Prohibición de Copias   | El plagio total o parcial conlleva calificación de 0.                         |
| Uso Responsable         | Las librerías externas deben referenciarse adecuadamente.                     |
| Revisión y Verificación | Los auxiliares podrán solicitar justificación del código en caso de sospecha. |

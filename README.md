# Proyecto 2: [GoDisk 2.0]

## 1. Resumen Ejecutivo

Este proyecto consiste en la evolución del sistema de archivos simulado del Proyecto 1, con el objetivo de mejorar su accesibilidad y usabilidad mediante una interfaz gráfica de usuario (GUI) basada en la web. El problema que se busca resolver es la dificultad de interactuar y visualizar la estructura del sistema de archivos únicamente a través de comandos de consola. Para solucionarlo, se desarrollará una aplicación web que permite explorar visualmente discos, particiones, carpetas y archivos de forma intuitiva.

Además de la visualización, se incorporarán nuevas funcionalidades como manejo del sistema de archivos EXT3, comandos avanzados (como fdisk, remove, edit, copy, chown, entre otros), y un sistema de journaling para registrar las operaciones. La solución estará desplegada en la nube utilizando servicios de AWS, con un frontend alojado en un bucket S3 y un backend en una instancia EC2 con sistema operativo Linux, asegurando así escalabilidad, accesibilidad y eficiencia.

## 2. Competencias que Desarrollaremos

• **Implementa un sistema de archivos con estructuras EXT2/EXT3** expuesto mediante una API e interfaz gráfica mediante la integración de funciones del backend y visualización de resultados en el frontend para gestionar operaciones relacionadas a la construcción y manejo de un sistema de archivos.

• **Integra soluciones de almacenamiento local y en la nube** mediante la selección y combinación de tecnologías de almacenamiento físico y virtualizado para diseñar infraestructuras eficientes, seguras y escalables.

• **Analiza arquitecturas de virtualización de almacenamiento** mediante la comparación de modelos y evaluación de retos operativos para optimizar la administración de recursos en entornos virtualizados.

## 3. Objetivos del Aprendizaje

### 3.1 Objetivo General

El estudiante será capaz de diseñar y desarrollar una aplicación web que permita la visualización y gestión de un sistema de archivos simulado, integrando tecnologías como Go para el backend y frameworks modernos para el frontend. La solución incluirá soporte para sistemas de archivos EXT2 y EXT3, nuevas funcionalidades de administración, y será desplegada en la nube utilizando servicios de AWS como EC2 y S3, aplicando conocimientos de programación, diseño de sistemas, y despliegue en entornos distribuidos.

### 3.2 Objetivos Específicos

Al finalizar el proyecto, los estudiantes deberán ser capaces de:

1. **Desarrollar un sistema funcional basado en la nube**
   Aplicar los conocimientos adquiridos en el curso para diseñar, implementar y probar una aplicación web que permita visualizar y gestionar un sistema de archivos simulado en formato EXT2 y EXT3. El sistema será desplegado en AWS, utilizando EC2 para el backend en Go y S3 para el frontend.
   
   *Ejemplo:* Los estudiantes podrán simular operaciones del sistema de archivos como creación de discos, montaje, navegación y recuperación desde una interfaz web accesible desde cualquier dispositivo.

2. **Diseñar interfaces gráficas funcionales e intuitivas**
   Implementar una interfaz gráfica de usuario (GUI) basada en tecnologías web modernas que permita una navegación sencilla del sistema de archivos, facilitando la interacción visual con discos, particiones, carpetas y archivos.
   
   *Ejemplo:* Los estudiantes podrán desarrollar una interfaz similar a un explorador de archivos que permita al usuario explorar visualmente la jerarquía del sistema y ejecutar comandos mediante una terminal embebida.

3. **Documentar adecuadamente el sistema desarrollado**
   Elaborar un manual técnico que incluya la arquitectura del sistema, los comandos implementados, la explicación de estructuras internas como journaling e inodos, y el proceso de despliegue en AWS.
   
   *Ejemplo:* Los estudiantes serán capaces de entregar un documento que describa detalladamente el funcionamiento interno del sistema de archivos y cómo fue implementado y desplegado en la nube.

## 4. Enunciado del Proyecto

En el mundo de la informática, las tecnologías y las necesidades de los usuarios están en constante evolución. Esto nos impulsa a mejorar y optimizar continuamente los sistemas existentes. En este proyecto, estamos evolucionando el Proyecto 1, haciéndolo más accesible y visualmente atractivo. Nuestro objetivo es desarrollar una interfaz gráfica de usuario (GUI) basada en web que permita visualizar fácilmente todo el sistema de archivos creado a través de comandos. Esto facilitará la navegación entre discos, particiones, carpetas y archivos. Además, se incorporarán nuevas funcionalidades que se detallarán más adelante y se ampliará el soporte para incluir sistemas de archivos EXT3.

Para asegurarnos de que todos puedan acceder al sistema y de que este pueda crecer fácilmente, hemos decidido usar la nube. Específicamente, estamos utilizando los servicios de Amazon Web Services (AWS). Aprovechando así sus capacidades de almacenamiento, procesamiento y despliegue de aplicaciones de manera eficiente y segura.

### 4.1 Descripción del Problema a Resolver

#### Arquitectura

El proyecto tendrá la siguiente arquitectura:

**Frontend**
La interfaz gráfica de usuario deberá ser desarrollada en una página web mediante un framework como React, Angular, Vue, u otro, dejando a discreción del estudiante el framework a utilizar.

Dicha página web deberá ser desplegada mediante el servicio de bucket S3 de AWS.

**Backend**
Para esta implementación se requerirá la creación de un backend, que será una API Rest desarrollada en lenguaje Go. Este backend deberá integrarse con el proyecto 1.

Dicho backend deberá ser desplegado en una instancia EC2 de AWS. Esta instancia deberá tener como sistema operativo alguna distribución Linux, se recomienda Ubuntu.

## Primera Parte (Frontend)

Para este apartado, se requiere un frontend con una página de inicio que incluya los siguientes elementos:

- **Área de Entrada de Comandos:** En este espacio, el usuario puede introducir manualmente los comandos a ejecutar o cargar un archivo de script que contenga los comandos.
  * **Botón de Carga de Archivo:** Permite seleccionar un archivo de script para cargar rápidamente los comandos en el área de entrada, facilitando su ejecución.
  * **Botón de Ejecutar:** Inicia la ejecución de todos los comandos en el área de entrada y muestra los mensajes generados por el servidor en el área de salida.

- **Explorador Visual del Sistema de Archivos:** Interfaz gráfica intuitiva que permite navegar visualmente por discos, particiones, carpetas y archivos creados.

## Segunda Parte (Backend)

En el backend, se implementará un sistema de archivos EXT2/EXT3. Para ello, la gestión de discos se simulará mediante archivos binarios con extensión .mia, en lugar de emplear dispositivos de almacenamiento físico. Estos archivos .mia almacenarán diversas estructuras que replicarán el funcionamiento del sistema de archivos, permitiendo la administración de múltiples particiones y simulando el comportamiento de un entorno de almacenamiento EXT2/EXT3.

### Funcionalidades Implementadas

- **Sistema de Archivos EXT2/EXT3** con soporte completo para journaling
- **Gestión de Usuarios y Permisos** con autenticación y autorización
- **Comandos Avanzados:** fdisk, mkfs, mount, unmount, mkfile, mkdir, remove, edit, copy, move, chown, chmod, cat, y más
- **Sistema de Journaling** para registro de operaciones en EXT3
- **API RESTful** para comunicación con el frontend
- **Reportes Estructurales** en formato gráfico y textual

### Información del Proyecto

**PONDERACIÓN:** 35  
**Horas Aproximadas:** 168  
**Curso:** Manejo e Implementación de Archivos (MIA)  

---

# MIA_2S2025_P2_202200389
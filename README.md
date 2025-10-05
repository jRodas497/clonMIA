# Proyecto 1:  [GoDisk]

El proyecto consiste en el desarrollo de una aplicación web multiplataforma llamada ExtreamFS, que permite simular y administrar un sistema de archivos basado en EXT2. La iniciativa busca abordar la dificultad de comprender e implementar Estructuras internas de sistemas de archivos reales, como discos, particiones, inodos y permisos, desde un enfoque práctico. Para ello, se crea una solución que integra un backend en Go y un frontend interactivo, permitiendo ejecutar comandos, gestionar usuarios, visualizar reportes estructurales y experimentar con funcionalidades clave de un sistema de archivos, todo de forma local y sin depender de hardware físico.

## Primer Parte (Frontend)
Para este apartado, se requiere un frontend con una página de inicio que incluya los siguientes elementos:
- Área de Entrada de Comandos: En este espacio, el usuario puede introducir manualmente los comandos a ejecutar o cargar un archivo de script que contenga los comandos.
    * Botón de Carga de Archivo: Permite seleccionar un archivo de script para cargar rápidamente los comandos en el área de entrada, facilitando su ejecución.
    * Botón de Ejecutar: Inicia la ejecución de todos los comandos en el área de entrada y muestra los mensajes generados por el servidor en el área de salida.

## Segunda Parte (Backend)
En el backend, se implementará un sistema de archivos EXT2. Para ello, la gestión de discos se simulará mediante archivos binarios con extensión .mia, en lugar de emplear dispositivos de almacenamiento físico. Estos archivos .mia almacenarán diversas Estructuras que replicarán el funcionamiento del sistema de archivos, permitiendo la administración de múltiples particiones y simulando el comportamiento de un entorno de almacenamiento EXT2.

### PONDERACIÓN: 35
### Horas Aproximadas: 168

# MIA_2S2025_P2_202200389

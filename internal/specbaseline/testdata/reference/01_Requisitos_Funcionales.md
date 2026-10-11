# Requisitos funcionales de la tienda demo

Documento sintético con las formas de una baseline real. Todo el texto es inventado.

## Catálogo

### RF-001: Alta de producto

El sistema DEBE permitir dar de alta un producto con nombre y precio.

#### Escenario de alta correcta

- **CUANDO** se envía un nombre y un precio válidos
- **ENTONCES** el producto queda registrado

#### Escenario de precio inválido

- **CUANDO** el precio es negativo
- **ENTONCES** el alta se rechaza

### RF-002: Consulta de producto

El sistema DEBE devolver los datos de un producto por su código.

### Notas de implantación del catálogo

Esta sección no es un requisito: describe la carga inicial del catálogo.

### RF-003: Baja de producto

El sistema DEBE permitir retirar un producto del catálogo.

```text
### RF-900: Requisito falso dentro de una valla
## Encabezado falso dentro de una valla
```

Después de la valla el bloque continúa.

### Criterios de aceptación generales

Sección independiente entre dos requisitos.

### Glosario del catálogo

Segunda sección contigua tras el requisito anterior.

### RF-040: Búsqueda por texto

El sistema DEBE permitir buscar productos por una palabra del nombre.

### RF-030: Orden de resultados

El sistema DEBE ordenar los resultados por relevancia (el ID desciende respecto al anterior).

## Pagos

### RF-020: Cobro con tarjeta

El sistema DEBE cobrar con tarjeta al confirmar el pedido.

### RF-021: Cobro con transferencia

El sistema DEBE admitir el pago por transferencia bancaria.

## Devoluciones

### RF-020: Devolución parcial

El sistema DEBE permitir devolver parte de un pedido (el ID se repite con otro título).

### RF-021: Devolución total

El sistema DEBE permitir devolver un pedido completo (el ID se repite con otro título).

#### Escenario de devolución fuera de plazo

- **CUANDO** han pasado más de 30 días
- **ENTONCES** la devolución se rechaza

## Informes

### RF-050: Informe mensual

El sistema DEBE generar un informe mensual de ventas.

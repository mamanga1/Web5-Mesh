# XionIA - Guía de Sesiones (Multitarea)

## 🎯 ¿Qué son las sesiones?

Las sesiones permiten **multitarea real** en XionIA. Podés tener un chat corriendo en background mientras trabajás en la bóveda. Cada sesión es un contexto aislado que podés crear, listar, activar y poner en segundo plano.

## 📋 Guía Paso a Paso

### 1. Ver sesiones activas

```
xion@nodo:~$ session list
📻 Sesiones activas:
  (ninguna)
```

### 2. Crear una nueva sesión de chat

```
xion@nodo:~$ session new chat did:maia:ABC123...
[CHAT] Modo conversación activado con did:maia:ABC123...
Escribí tu mensaje o 'session detach' para salir.
```

La pantalla cambia al modo `[CHAT]`. Todo lo que escribás se envía cifrado al nodo destino.

### 3. Enviar mensajes

```
[CHAT] > Hola, ¿cómo estás?
✅ Mensaje enviado
[CHAT] > ¿Viste el nuevo commit?
✅ Mensaje enviado
```

### 4. Poner la sesión en background

```
[CHAT] > session detach
📻 Sesión puesta en background.
xion@nodo:~$
```

Volvés al inicio, pero **el chat sigue vivo** en background.

### 5. Trabajar en la bóveda mientras el chat corre

```
xion@nodo:~$ /ls
📂 Contenido de tu espacio de trabajo:
  📄 documento.pdf

xion@nodo:~$ verify documento.pdf
✅ VERIFICACIÓN EXITOSA
```

### 6. Volver a la sesión de chat

```
xion@nodo:~$ session attach 1
[CHAT] Conectado a sesión con did:maia:ABC123...
```

### 7. Salir de la sesión

```
[CHAT] > session detach
📻 Sesión puesta en background.
```

O si querés cerrar la sesión completamente, salí de la shell con `exit`.

## 📻 Tipos de Sesiones

| Tipo | Descripción | Comando |
|:-----|:------------|:--------|
| **chat** | Conversación E2E con otro nodo | `session new chat <did>` |

## 💡 Tips

- **Múltiples sesiones:** Podés tener varias sesiones activas al mismo tiempo.
- **IDs numéricos:** Cada sesión tiene un ID (1, 2, 3...) que usás con `session attach`.
- **Persistencia:** Las sesiones viven mientras la shell esté abierta. Al hacer `exit`, se cierran todas.
- **Cifrado E2E:** Todas las sesiones están cifradas de punta a punta. El Faro solo ve blobs.

## 🎬 Ejemplo Completo

```
# Iniciar shell
./mesh shell

# Crear sesión de chat
session new chat did:maia:ABC123...

# Enviar mensajes
[CHAT] > Hola
[CHAT] > ¿Tenés el archivo?
[CHAT] > session detach

# Trabajar en la bóveda
xion@nodo:~$ sign documento.pdf
✅ Archivo firmado

# Volver al chat
xion@nodo:~$ session attach 1
[CHAT] > Te lo firmé, verificá con verify
[CHAT] > session detach

# Salir
xion@nodo:~$ exit
```

---

*Guía de sesiones de XionIA - La Xión Digital 🦾*
```

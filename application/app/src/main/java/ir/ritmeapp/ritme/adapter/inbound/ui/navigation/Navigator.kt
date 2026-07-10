package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.remember

/**
 * A tiny hand-written back stack (CLAUDE.md §3 — no navigation library). Backed by a
 * Compose snapshot list so pushes/pops recompose the host automatically. The bottom of the
 * stack is the root/start destination and is never popped.
 */
@Stable
class Navigator(start: Destination) {

    private val _backStack = mutableStateListOf(start)
    val backStack: List<Destination> get() = _backStack

    val current: Destination get() = _backStack.last()
    val canPop: Boolean get() = _backStack.size > 1

    fun push(destination: Destination) {
        _backStack.add(destination)
    }

    /** Pops the top destination; returns false if already at the root. */
    fun pop(): Boolean {
        if (!canPop) return false
        _backStack.removeAt(_backStack.lastIndex)
        return true
    }

    /** Clears the stack down to a single [destination] — used by crash recovery. */
    fun replaceAll(destination: Destination) {
        _backStack.clear()
        _backStack.add(destination)
    }
}

@Composable
fun rememberNavigator(start: Destination): Navigator = remember { Navigator(start) }

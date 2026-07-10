package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.HomeService
import ir.ritmeapp.ritme.domain.model.ServiceAccent
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The post-login home: a personal greeting plus the Ritme services list. The list is a
 * [LazyColumn] keyed by service id (CLAUDE.md §5). Records the last-safe-screen on entry
 * (§7.2). Being the post-auth root, it is not swipe-back-wrapped.
 */
@Composable
fun HomeScreen(
    onServiceSelected: (HomeService) -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: HomeViewModel = viewModel(factory = container.homeViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:home:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Home.route, null, System.currentTimeMillis()),
        )
    }

    HomeContent(state = state, onServiceSelected = onServiceSelected, modifier = modifier)
}

@Composable
private fun HomeContent(
    state: HomeUiState,
    onServiceSelected: (HomeService) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Scaffold(modifier = modifier.fillMaxSize(), containerColor = colors.background) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 24.dp),
        ) {
            item(key = "header") { Header(state, colors) }
            item(key = "section") {
                Spacer(Modifier.height(24.dp))
                Text(
                    text = stringResource(R.string.home_services_title),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                )
                Spacer(Modifier.height(12.dp))
            }
            items(state.services, key = { it.id }) { service ->
                ServiceCard(service, colors, onServiceSelected)
                Spacer(Modifier.height(12.dp))
            }
        }
    }
}

@Composable
private fun Header(state: HomeUiState, colors: RitmeColors) {
    val greeting = state.greetingName
        ?.let { stringResource(R.string.home_greeting_named, it) }
        ?: stringResource(R.string.home_greeting)
    Column {
        Text(
            text = greeting,
            style = MaterialTheme.typography.headlineMedium,
            color = colors.ink,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            text = stringResource(R.string.home_subtitle),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
        )
    }
}

@Composable
private fun ServiceCard(
    service: HomeService,
    colors: RitmeColors,
    onSelected: (HomeService) -> Unit,
) {
    val accent = service.accent.color(colors)
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(colors.surface)
            .clickable { onSelected(service) }
            .padding(16.dp),
    ) {
        Column(Modifier.fillMaxWidth()) {
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(accent.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = service.title.take(1),
                    style = MaterialTheme.typography.titleMedium,
                    color = accent,
                    textAlign = TextAlign.Center,
                )
            }
            Spacer(Modifier.height(12.dp))
            Text(
                text = service.title,
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = service.tagline,
                style = MaterialTheme.typography.bodySmall,
                color = colors.inkMuted,
            )
        }
    }
}

private fun ServiceAccent.color(colors: RitmeColors) = when (this) {
    ServiceAccent.Pink -> colors.pink
    ServiceAccent.Accent -> colors.accent
    ServiceAccent.Info -> colors.info
}

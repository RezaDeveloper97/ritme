package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatFull
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.AppNotification
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The notifications inbox (web `/profile/notifications`): unread-highlighted rows with relative
 * Jalali time labels; tapping an unread row marks it read, «خواندن همه» clears everything.
 */
@Composable
fun NotificationsScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: NotificationsViewModel = viewModel(factory = container.notificationsViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:notifications:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfileNotifications.route, null, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = {
            ScreenHeader(
                title = stringResource(R.string.notifications_title),
                onBack = onBack,
                trailing = {
                    if ((state.page?.unreadCount ?: 0) > 0) {
                        Text(
                            text = stringResource(R.string.notifications_mark_all),
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.pink,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier
                                .clip(RoundedCornerShape(10.dp))
                                .clickable { viewModel.onIntent(NotificationsIntent.MarkAllRead) }
                                .padding(horizontal = 8.dp, vertical = 6.dp),
                        )
                    }
                },
            )
        },
    ) { padding ->
        val items = state.page?.items.orEmpty()
        when {
            state.loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.pink)
            }

            items.isEmpty() -> EmptyInbox(colors, Modifier.padding(padding))

            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(items, key = { it.id }) { notification ->
                    NotificationRow(
                        notification = notification,
                        onTap = { viewModel.onIntent(NotificationsIntent.MarkRead(notification.id)) },
                        colors = colors,
                    )
                }
            }
        }
    }
}

@Composable
private fun EmptyInbox(colors: RitmeColors, modifier: Modifier = Modifier) {
    Column(
        modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_bell),
            contentDescription = null,
            tint = colors.pink,
            modifier = Modifier.size(36.dp),
        )
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.notifications_empty_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            stringResource(R.string.notifications_empty_body),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun NotificationRow(notification: AppNotification, onTap: () -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(if (notification.isRead) colors.surface else colors.pinkContainer)
            .clickable(enabled = !notification.isRead, onClick = onTap)
            .padding(14.dp),
        verticalAlignment = Alignment.Top,
    ) {
        if (!notification.isRead) {
            Box(
                Modifier
                    .padding(top = 6.dp)
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(colors.pink),
            )
            Spacer(Modifier.width(8.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(
                text = notification.title,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = if (notification.isRead) FontWeight.Normal else FontWeight.Bold,
            )
            if (notification.body.isNotBlank()) {
                Spacer(Modifier.height(2.dp))
                Text(
                    text = notification.body,
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                )
            }
            Spacer(Modifier.height(4.dp))
            Text(
                text = relativeTimeLabel(notification.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
        }
    }
}

/** «امروز» / «دیروز» / «n روز پیش» / a full Jalali date past a week — like the web inbox. */
@Composable
private fun relativeTimeLabel(createdAtIso: String): String {
    val date = remember(createdAtIso) {
        createdAtIso.substringBefore('T').substringBefore(' ').let(GregorianDate::parseIso)?.toJalali()
    } ?: return ""
    val daysAgo = todayJalali().toJdn() - date.toJdn()
    return when {
        daysAgo <= 0 -> stringResource(R.string.notifications_today)
        daysAgo == 1 -> stringResource(R.string.notifications_yesterday)
        daysAgo <= DAYS_IN_WEEK -> stringResource(R.string.notifications_days_ago, daysAgo.toPersianDigits())
        else -> date.formatFull()
    }
}

private const val DAYS_IN_WEEK = 7

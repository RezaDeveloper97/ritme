package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/**
 * The web `.chip`: a rounded outlined pill that fills soft pink and turns brand-colored when
 * selected. Used for single- and multi-select option groups across the log and pregnancy forms.
 */
@Composable
fun SelectableChip(
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(22.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(
                width = 1.5.dp,
                color = if (selected) colors.pink else colors.outline,
                shape = RoundedCornerShape(22.dp),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 9.dp),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = if (selected) colors.pink else colors.ink,
            fontWeight = FontWeight.Bold,
        )
    }
}

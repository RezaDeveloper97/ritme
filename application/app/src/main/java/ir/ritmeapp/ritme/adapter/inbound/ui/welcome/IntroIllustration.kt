package ir.ritmeapp.ritme.adapter.inbound.ui.welcome

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors

/** The three intro slides' artwork, kept as a closed set so the carousel picks one exhaustively. */
enum class IntroArt { RHYTHM, TRACK, INSIGHT }

/**
 * Hand-drawn, minimal illustrations for the welcome slides — pure Compose [Canvas] (CLAUDE.md §3,
 * no Lottie/SVG library), tinted from the brand palette so they track light/dark automatically.
 * Each slide passes its own [accentFrom]/[accentTo] so the primary accent is a per-slide diagonal
 * gradient (mirrors the web `IntroIllustration` which swaps only the accent gradient per slide).
 */
@Composable
fun IntroIllustration(
    art: IntroArt,
    accentFrom: Color,
    accentTo: Color,
    modifier: Modifier = Modifier,
) {
    val line = LocalRitmeColors.current.outline
    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(1.15f),
    ) {
        val accent = Brush.linearGradient(
            colors = listOf(accentFrom, accentTo),
            start = Offset.Zero,
            end = Offset(size.width, size.height),
        )
        val soft = accentTo.copy(alpha = 0.12f)
        when (art) {
            IntroArt.RHYTHM -> drawRhythm(accent, accentTo, soft, line)
            IntroArt.TRACK -> drawTrack(accent, accentTo, soft, line)
            IntroArt.INSIGHT -> drawInsight(accent, accentTo, soft, line)
        }
    }
}

/** Concentric "cycle" rings with a body-rhythm wave and a moon/sun pair. */
private fun DrawScope.drawRhythm(accent: Brush, secondary: Color, soft: Color, line: Color) {
    val c = center
    val r = size.minDimension * 0.36f
    drawCircle(soft, radius = r * 1.25f, center = c)
    drawCircle(accent, radius = r, center = c, style = Stroke(width = r * 0.06f))
    drawCircle(line, radius = r * 0.62f, center = c, style = Stroke(width = r * 0.05f))

    val wave = Path().apply {
        val w = r * 1.6f
        val startX = c.x - w / 2
        moveTo(startX, c.y)
        val step = w / 4
        var x = startX
        var up = true
        while (x < c.x + w / 2) {
            quadraticBezierTo(x + step / 2, c.y + (if (up) -r * 0.4f else r * 0.4f), x + step, c.y)
            x += step
            up = !up
        }
    }
    drawPath(wave, accent, style = Stroke(width = r * 0.05f))
    drawCircle(accent, radius = r * 0.1f, center = Offset(c.x - r, c.y - r))
    drawCircle(secondary, radius = r * 0.1f, center = Offset(c.x + r, c.y - r * 0.8f))
}

/** A small calendar grid with one highlighted day and a mood dot row beneath. */
private fun DrawScope.drawTrack(accent: Brush, secondary: Color, soft: Color, line: Color) {
    val gridW = size.width * 0.6f
    val gridH = size.height * 0.5f
    val left = (size.width - gridW) / 2
    val top = size.height * 0.16f
    val cols = 5
    val rows = 4
    val cell = minOf(gridW / cols, gridH / rows)
    drawRoundRect(
        soft,
        topLeft = Offset(left - cell * 0.3f, top - cell * 0.3f),
        size = Size(cell * cols + cell * 0.6f, cell * rows + cell * 0.6f),
        cornerRadius = CornerRadius(cell * 0.4f),
    )
    for (row in 0 until rows) {
        for (col in 0 until cols) {
            val cx = left + col * cell + cell / 2
            val cy = top + row * cell + cell / 2
            val highlighted = row == 1 && col == 2
            if (highlighted) {
                drawCircle(accent, radius = cell * 0.34f, center = Offset(cx, cy))
            } else {
                drawCircle(secondary.copy(alpha = 0.28f), radius = cell * 0.22f, center = Offset(cx, cy))
            }
        }
    }
    val moodY = top + rows * cell + cell
    drawCircle(accent, radius = cell * 0.3f, center = Offset(left + cell, moodY))
    drawCircle(secondary, radius = cell * 0.3f, center = Offset(left + cell + cell * 1.5f, moodY))
    drawCircle(line, radius = cell * 0.3f, center = Offset(left + cell + cell * 3f, moodY))
}

/** A rising insight line over a soft panel with an accent "spark" node. */
private fun DrawScope.drawInsight(accent: Brush, secondary: Color, soft: Color, line: Color) {
    val panelLeft = size.width * 0.16f
    val panelTop = size.height * 0.2f
    val panelW = size.width * 0.68f
    val panelH = size.height * 0.5f
    drawRoundRect(
        soft,
        topLeft = Offset(panelLeft, panelTop),
        size = Size(panelW, panelH),
        cornerRadius = CornerRadius(panelH * 0.16f),
    )
    // Faint baseline grid.
    val gridColor = line.copy(alpha = 0.6f)
    for (fy in listOf(0.35f, 0.65f, 0.95f)) {
        val y = panelTop + fy * panelH
        drawLine(gridColor, Offset(panelLeft, y), Offset(panelLeft + panelW, y), strokeWidth = panelH * 0.012f)
    }
    val points = listOf(0.0f to 0.7f, 0.25f to 0.5f, 0.5f to 0.62f, 0.75f to 0.3f, 1.0f to 0.18f)
    val path = Path()
    points.forEachIndexed { i, (fx, fy) ->
        val x = panelLeft + fx * panelW
        val y = panelTop + fy * panelH
        if (i == 0) path.moveTo(x, y) else path.lineTo(x, y)
    }
    drawPath(path, accent, style = Stroke(width = panelH * 0.05f))
    points.forEach { (fx, fy) ->
        drawCircle(accent, radius = panelH * 0.04f, center = Offset(panelLeft + fx * panelW, panelTop + fy * panelH))
    }
    // AI spark: a four-point star node near the peak.
    val sx = panelLeft + panelW
    val sy = panelTop + panelH * 0.18f
    val s = panelH * 0.16f
    val star = Path().apply {
        moveTo(sx, sy - s)
        lineTo(sx + s * 0.28f, sy - s * 0.28f)
        lineTo(sx + s, sy)
        lineTo(sx + s * 0.28f, sy + s * 0.28f)
        lineTo(sx, sy + s)
        lineTo(sx - s * 0.28f, sy + s * 0.28f)
        lineTo(sx - s, sy)
        lineTo(sx - s * 0.28f, sy - s * 0.28f)
        close()
    }
    drawPath(star, accent)
}

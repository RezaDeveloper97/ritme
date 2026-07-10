# Keep line numbers for readable crash traces, strip source file names.
-keepattributes SourceFile,LineNumberTable
-renamesourcefileattribute SourceFile

# Compose + Kotlin metadata are handled by the bundled AGP/Compose rules.
# Add app-specific keep rules below as needed (e.g. reflection, serialization).

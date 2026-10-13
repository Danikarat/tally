package com.danikarat.tally

import android.app.Application
import dagger.hilt.android.HiltAndroidApp

/**
 * TallyApplication is the base Application class for Tally.
 * Annotated with @HiltAndroidApp to trigger Hilt code generation and dependency container initialization.
 */
@HiltAndroidApp
class TallyApplication : Application()

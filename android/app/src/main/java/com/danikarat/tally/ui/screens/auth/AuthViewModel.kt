package com.danikarat.tally.ui.screens.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.danikarat.tally.data.remote.model.AuthResponse
import com.danikarat.tally.data.repository.AuthRepository
import com.danikarat.tally.ui.common.UiState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * AuthViewModel manages UI state and interactions for Login and Registration flows.
 */
@HiltViewModel
class AuthViewModel @Inject constructor(
    private val authRepository: AuthRepository
) : ViewModel() {

    private val _authState = MutableStateFlow<UiState<AuthResponse>>(UiState.Idle)
    val authState: StateFlow<UiState<AuthResponse>> = _authState.asStateFlow()

    fun login(email: String, pass: String) {
        val emailError = AuthValidator.validateEmail(email)
        if (emailError != null) {
            _authState.value = UiState.Error(emailError)
            return
        }

        if (pass.isBlank()) {
            _authState.value = UiState.Error("Password cannot be empty")
            return
        }

        _authState.value = UiState.Loading
        viewModelScope.launch {
            val result = authRepository.login(email, pass)
            result.onSuccess { authResponse ->
                _authState.value = UiState.Success(authResponse)
            }.onFailure { error ->
                _authState.value = UiState.Error(error.message ?: "Authentication failed")
            }
        }
    }

    fun register(email: String, pass: String, fullName: String) {
        val nameError = AuthValidator.validateFullName(fullName)
        if (nameError != null) {
            _authState.value = UiState.Error(nameError)
            return
        }

        val emailError = AuthValidator.validateEmail(email)
        if (emailError != null) {
            _authState.value = UiState.Error(emailError)
            return
        }

        val passError = AuthValidator.validatePassword(pass)
        if (passError != null) {
            _authState.value = UiState.Error(passError)
            return
        }

        _authState.value = UiState.Loading
        viewModelScope.launch {
            val result = authRepository.register(email, pass, fullName)
            result.onSuccess { authResponse ->
                _authState.value = UiState.Success(authResponse)
            }.onFailure { error ->
                _authState.value = UiState.Error(error.message ?: "Registration failed")
            }
        }
    }

    fun resetState() {
        _authState.value = UiState.Idle
    }
}

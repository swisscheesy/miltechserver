import { signInWithEmailAndPassword, deleteUser, type User } from 'firebase/auth';
import { auth } from '../firebase';

export interface AuthResult {
  success: boolean;
  message: string;
  user?: User;
}

/**
 * Authenticates user with email and password
 */
export async function authenticateUser(email: string, password: string): Promise<AuthResult> {
  try {
    const userCredential = await signInWithEmailAndPassword(auth, email, password);
    return {
      success: true,
      message: 'Authentication successful',
      user: userCredential.user
    };
  } catch (error: any) {
    let message = 'Authentication failed';
    
    // Handle specific Firebase auth errors
    switch (error.code) {
      case 'auth/user-not-found':
        message = 'No account found with this email address';
        break;
      case 'auth/wrong-password':
        message = 'Incorrect password';
        break;
      case 'auth/invalid-email':
        message = 'Invalid email address';
        break;
      case 'auth/user-disabled':
        message = 'This account has been disabled';
        break;
      case 'auth/too-many-requests':
        message = 'Too many failed attempts. Please try again later';
        break;
      case 'auth/invalid-credential':
        message = 'Invalid email or password';
        break;
      default:
        message = `Authentication error: ${error.message}`;
    }
    
    return {
      success: false,
      message
    };
  }
}

/**
 * Deletes the currently authenticated user's account
 */
export async function deleteUserAccount(user: User): Promise<AuthResult> {
  try {
    await deleteUser(user);
    return {
      success: true,
      message: 'Your account has been successfully deleted'
    };
  } catch (error: any) {
    let message = 'Failed to delete account';
    
    // Handle specific Firebase errors
    switch (error.code) {
      case 'auth/requires-recent-login':
        message = 'For security reasons, please log in again and try deleting your account';
        break;
      default:
        message = `Account deletion error: ${error.message}`;
    }
    
    return {
      success: false,
      message
    };
  }
}

/**
 * Complete account deletion process: authenticate then delete
 */
export async function deleteAccountWithCredentials(email: string, password: string): Promise<AuthResult> {
  // First authenticate the user
  const authResult = await authenticateUser(email, password);
  
  if (!authResult.success || !authResult.user) {
    return authResult;
  }
  
  // If authentication successful, delete the account
  return await deleteUserAccount(authResult.user);
} 
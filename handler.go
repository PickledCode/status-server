package main

// HandleClient provides the client access to the database
// through a message-based API.
//
// This automatically closes the connection.
func HandleClient(conn Connection, db EventDB) {
	defer conn.Close()
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch msg := msg.(type) {
		case *LoginMessage:
			if sess, err := db.BeginSession(msg.Email, msg.Password); err != nil {
				err = conn.WriteMessage(&LoginFailureMessage{Message: err.Error()})
				if err != nil {
					return
				}
			} else {
				err = conn.WriteMessage(&LoginSuccessMessage{})
				if err != nil {
					return
				}
				handleAuthenticated(conn, db, sess)
				return
			}
		case *RegisterMessage:
			var resMessage Message
			if err := db.AddUser(msg.Email, msg.Password); err != nil {
				resMessage = &RegisterFailureMessage{Message: err.Error()}
			} else {
				resMessage = &RegisterSuccessMessage{}
			}
			if err := conn.WriteMessage(resMessage); err != nil {
				return
			}
		case *RegisterVerifyMessage:
			// TODO: this.
		case *ResetPasswordMessage:
			// TODO: this.
		}
	}
}

func handleAuthenticated(conn Connection, db EventDB, sess DBSession) {
	defer sess.Close()
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for {
			select {
			case <-stopChan:
				return
			default:
			}
			select {
			case <-stopChan:
				return
			case <-sess.Events():
				// TODO: turn event into message & send it.
			}
		}
	}()

	defer func() {
		close(stopChan)
		<-doneChan
	}()

	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		switch msg := msg.(type) {
		case *LogoutMessage:
			// TODO: should we just get rid of this silly API?
			return
		case *LogoutOtherMessage:
			if err := sess.DisconnectOthers(); err != nil {
				// TODO: write error here.
			}
		case *SetStatusMessage:
			if err := sess.SetStatus(msg.UserStatus); err != nil {
				// TODO: write error here.
			}
		default:
			return
			// TODO: lots of other handlers here.
		}
	}
}

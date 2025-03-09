package gpcm

var (
	WWFCMsgUnknownLoginError = WWFCErrorMessage{
		ErrorCode: 22000,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCへの ログイン中に\n" +
				"不明なエラー が発生しました\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"An unknown error has occurred\n" +
				"while logging in to Retro WFC.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Ein unbekannter Fehler ist beim\n" +
				"Verbinden mit Retro WFC aufgetreten.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Un error desconocido ha ocurrido\n" +
				"al conectarse a Retro WFC.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"È stato riscontrato un errore sconosciuto\n" +
				"durante l'accesso alla Retro WFC.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Er is een onbekende fout opgetreden\n" +
				"tijdens het verbinden met Retro WFC.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Une erreur inconnue s'est produite\n" +
				"pendant la connexion à Retro WFC.\n" +
				"\n" +
				"Code Erreur:  %[1]d",
		},
	}

	WWFCMsgDolphinSetupRequired = WWFCErrorMessage{
		ErrorCode: 22001,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Dolphinで Retro WFCに接続するには\n" +
				"ついかの セットアップが必要です\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"Additional setup is required\n" +
				"to use Retro WFC on Dolphin.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Für die Verwendung von Retro WFC auf Dolphin\n" +
				"ist eine zusätzliche Einrichtung erforderlich.\n." +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Se requiere una instalación adicional\n" +
				"para poder usar Retro WFC en Dolphin.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Un'ulteriore installazione è necessaria\n" +
				"per usare la Retro WFC su Dolphin.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Extra instellingen zijn vereist\n" +
				"om Retro WFC op Dolphin te gebruiken.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Une installation additionnelle est requise\n" +
				"pour utiliser Retro WFC sur Dolphin\n." +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgProfileBannedTOS = WWFCErrorMessage{
		ErrorCode: 22002,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"利用きやくに いはんしたため\n" +
				"Retro WFCから BANされました\n" +
				"\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You are banned from Retro WFC\n" +
				"due to a violation of the\n" +
				"Terms of Service.\n" +
				"\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von Retro WFC\n" +
				"wegen eines Verstoßes der\n" +
				"Terms of Service gebannt.\n" +
				"\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangSpanish: "" +
				"Te han baneado de Retro WFC\n" +
				"debido a una violación de los\n" +
				"Terminos de Servicio.\n" +
				"\n" +
				"Código de Error: %[1]d\n" +
				"Información de soporte: NG%08[2]x",
			LangItalian: "" +
				"Sei stato bannato dalla Retro WFC\n" +
				"a causa di una violazione dei\n" +
				"Termini di Servizio.\n" +
				"\n" +
				"Codice Errore: %[1]d\n" +
				"Supporto Informativo: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van Retro WFC\n" +
				"vanwege een overtreding van de\n" +
				"gebruiksvoorwaarden.\n" +
				"\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous avez été banni(e) de Retro WFC" +
				"à cause d'une violation des" +
				"Conditions de Service" +
				"\n" +
				"Code Erreur:  %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgProfileBannedTOSNow = WWFCErrorMessage{
		ErrorCode: 22002,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"利用きやくに いはんしたため\n" +
				"Retro WFCから BANされています\n" +
				"\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You are banned from Retro WFC\n" +
				"due to a violation of the\n" +
				"Terms of Service.\n" +
				"\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von Retro WFC\n" +
				"wegen eines Verstoßes der\n" +
				"Terms of Service gebannt.\n" +
				"\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangSpanish: "" +
				"Te han baneado de Retro WFC\n" +
				"debido a una violación de los\n" +
				"Terminos de Servicio.\n" +
				"\n" +
				"Código de Error: %[1]d\n" +
				"Información de soporte: NG%08[2]x",
			LangItalian: "" +
				"Sei stato bannato dalla\n" +
				"Retro WFC a causa di una violazione\n" +
				"dei Termini di Servizio.\n" +
				"\n" +
				"Codice Errore: %[1]d\n" +
				"Supporto Informativo: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van Retro WFC\n" +
				"vanwege een overtreding van de\n" +
				"gebruiksvoorwaarden.\n" +
				"\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous avez été banni(e) de Retro WFC" +
				"à cause d'une violation des" +
				"Conditions de Service" +
				"\n" +
				"Code Erreur:  %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgProfileRestricted = WWFCErrorMessage{
		ErrorCode: 22003,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCの ルールにいはんしたため\n" +
				"オンライン対戦から BANされました\n" +
				"\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You are banned from public\n" +
				"matches due to a violation\n" +
				"of the Retro WFC Rules.\n" +
				"\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von öffentl. Räumen\n" +
				"wegen eines Verstoßes der\n" +
				"Retro WFC Regeln gebannt.\n" +
				"\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangSpanish: "" +
				"Te han baneado de partidas públicas\n" +
				"debido a una violación de las\n" +
				"reglas de Retro WFC.\n" +
				"\n" +
				"Código de Error: %[1]d\n" +
				"Información de soporte: NG%08[2]x",
			LangItalian: "" +
				"Sei stato bannato dalle corse\n" +
				"pubbliche a causa di una violazione\n" +
				"delle regole della Retro WFC.\n" +
				"\n" +
				"Codice Errore: %[1]d\n" +
				"Supporto Informativo: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van openbare\n" +
				"wedstrijden vanwege een overtreding\n" +
				"van de Retro WFC-regels.\n" +
				"\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous avez été banni(e) des matchs\n" +
				"public à cause d'un violation d'une\n" +
				"des règles de Retro WFC" +
				"\n" +
				"Code Erreur:  %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgProfileRestrictedNow = WWFCErrorMessage{
		ErrorCode: 22003,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCの ルールにいはんしたため\n" +
				"オンライン対戦から BANされています\n" +
				"\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You have been banned from public\n" +
				"matches due to a violation\n" +
				"of the Retro WFC Rules.\n" +
				"\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von öffentl. Räumen\n" +
				"wegen eines Verstoßes der\n" +
				"Retro WFC Regeln gebannt.\n" +
				"\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangSpanish: "" +
				"Te han baneado de partidas públicas\n" +
				"debido a una violación de las\n" +
				"reglas de Retro WFC.\n" +
				"\n" +
				"Código de Error: %[1]d\n" +
				"Información de soporte: NG%08[2]x",
			LangItalian: "" +
				"Sei stato bannato dalle corse\n" +
				"pubbliche a causa di una violazione\n" +
				"delle regole della Retro WFC.\n" +
				"\n" +
				"Codice Errore: %[1]d\n" +
				"Supporto Informativo: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van openbare\n" +
				"wedstrijden vanwege een overtreding\n" +
				"van de Retro WFC-regels.\n" +
				"\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous avez été banni(e) des matchs\n" +
				"publics à cause d'un violation d'une\n" +
				"des règles de Retro WFC" +
				"\n" +
				"Code Erreur:  %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgProfileRestrictedCustom = WWFCErrorMessage{
		ErrorCode: 22003,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"オンライン対戦から BANされました\n" +
				"りゆう： %[3]s\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You are banned from public matches.\n" +
				"Reason: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von öffentlichen Matches ausgeschlossen.\n" +
				"Grund: %[3]s\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van openbare wedstrijden.\n" +
				"Reden: %[3]s\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous êtes banni(e) des matchs publics.\n" +
				"Raison: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgProfileRestrictedNowCustom = WWFCErrorMessage{
		ErrorCode: 22003,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"オンライン対戦から BANされました\n" +
				"りゆう： %[3]s\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You are banned from public matches.\n" +
				"Reason: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von öffentlichen Matches ausgeschlossen.\n" +
				"Grund: %[3]s\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van openbare wedstrijden.\n" +
				"Reden: %[3]s\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous êtes banni(e) des matchs publics.\n" +
				"Raison: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgKickedGeneric = WWFCErrorMessage{
		ErrorCode: 22004,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCから キックされました\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"You have been kicked from\n" +
				"Retro WFC.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Du wurdest aus Retro WFC\n" +
				"gekickt.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Te han expulsado de Retro WFC\n" +
				"\n" +
				"Código de Error: %[1]d",
			LangItalian: "" +
				"Sei stato espulso\n" +
				"dalla Retro WFC.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Je bent uit Retro WFC\n" +
				"geschopt.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Vous avez été expulsé de\n" +
				"Retro WFC." +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgKickedModerator = WWFCErrorMessage{
		ErrorCode: 22004,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCの モデレーターから\n" +
				"キックされました\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"You have been kicked from\n" +
				"Retro WFC by a moderator.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Du wurdest von einem Moderator\n" +
				"aus Retro WFC gekickt.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Un moderador te ha\n" +
				"expulsado de Retro WFC\n" +
				"\n" +
				"Código de Error: %[1]d",
			LangItalian: "" +
				"Sei stato espulso dalla\n" +
				"Retro WFC da un moderatore.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Je bent uit Retro WFC\n" +
				"geschopt door een moderator.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Vous avez été expulsé de\n" +
				"Retro WFC par un modérateur.\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgKickedRoomHost = WWFCErrorMessage{
		ErrorCode: 22004,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"フレンドルームの ホストから\n" +
				"キックされました\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"You have been kicked from the\n" +
				"friend room by the room creator.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Du wurdest von dem Raum-Ersteller\n" +
				"aus der Freundes-Lobby gekickt.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"El creador de la sala te ha\n" +
				"expulsado de ella\n" +
				"\n" +
				"Código de Error: %[1]d",
			LangItalian: "" +
				"Sei stato espulso dalla\n" +
				"stanza dal suo creatore.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Je bent uit de vriendenkamer\n" +
				"geschopt door de gastheer.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Vous avez été expulsé de la salle\n" +
				"par le créateur." +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgKickedCustom = WWFCErrorMessage{
		ErrorCode: 22004,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"オンライン対戦から BANされています\n" +
				"りゆう： %[3]s\n" +
				"エラーコード： %[1]d\n" +
				"サポート情報： NG%08[2]x",
			LangEnglish: "" +
				"You have been banned from public matches,\n" +
				"Reason: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Support Info: NG%08[2]x",
			LangGerman: "" +
				"Du wurdest von öffentlichen Matches ausgeschlossen,\n" +
				"Grund: %[3]s\n" +
				"Fehlercode: %[1]d\n" +
				"Support-Info: NG%08[2]x",
			LangDutch: "" +
				"Je bent verbannen van openbare wedstrijden,\n" +
				"Reden: %[3]s\n" +
				"Foutcode: %[1]d\n" +
				"Ondersteuningsinformatie: NG%08[2]x",
			LangFrenchEU: "" +
				"Vous avez été banni(e) des matchs publics.\n" +
				"Raison: %[3]s\n" +
				"Error Code: %[1]d\n" +
				"Information Support: NG%08[2]x",
		},
	}

	WWFCMsgConsoleMismatch = WWFCErrorMessage{
		ErrorCode: 22005,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"使われているコンソールは \n" +
				"このプロファイルが 登録されたときに\n" +
				"使われたコンソールでは ありません\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"The console you are using is not\n" +
				"the device used to register this\n" +
				"profile.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Die Konsole die du gerade nutzt\n" +
				"ist nicht die selbe mit der dieses\n" +
				"Profil erstellt wurde.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"La consola que estas usando no es el\n" +
				"dispositivo usado para registrar este\n" +
				"perfil\n" +
				"\n" +
				"Código de Error: %[1]d",
			LangItalian: "" +
				"La console che stai usando non è\n" +
				"il dispositivo usato per\n" +
				"registrare questo profilo.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"De console die je gebruikt is niet\n" +
				"het apparaat dat is gebruikt om dit\n" +
				"profiel te registreren.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"La console que vous utilisé\n" +
				"n'est pas l'appareil utilisé pour\n" +
				"enregistrer ce profil." +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgConsoleMismatchDolphin = WWFCErrorMessage{
		ErrorCode: 22005,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"使われているコンソールは \n" +
				"このプロファイルが 登録されたときに\n" +
				"使われたコンソールでは ありません\n" +
				"NANDがただしく 設定されていることを\n" +
				"ご確認ください\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"The console you are using is not\n" +
				"the device used to register this\n" +
				"profile. Please make sure you've\n" +
				"set up your NAND correctly.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Die Konsole die du gerade nutzt\n" +
				"ist nicht die selbe mit der dieses\n" +
				"Profil erstellt wurde. Bitte gehe sicher\n" +
				"dass du das NAND korrekt initialisiert hast\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"La consola que estas usando no es el\n" +
				"dispositivo usado para registrar este\n" +
				"perfil. Asegurate que has configurado\n" +
				"correctamente tu NAND.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"La console che stai usando non è\n" +
				"il dispositivo usato per registrare\n" +
				"questo profilo. Assicurati di avere\n" +
				"impostato la tua NAND correttamente.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"De console die je gebruikt is niet\n" +
				"het apparaat dat is gebruikt om dit\n" +
				"profiel te registreren. Zorg ervoor dat de\n" +
				"NAND juist is ingesteld.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"La consonle que vous utilisé\n" +
				"n'est pas l'appareil utilisé pour\n" +
				"enregistrer ce profil. Assurez-vous d'avoir\n" +
				"configuré votre NAND correctement.\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgProfileIDInvalid = WWFCErrorMessage{
		ErrorCode: 22006,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"あなたが登録しようとした \n" +
				"プロファイルIDは むこうです\n" +
				"新しくライセンスをつくりなおしてください\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"The profile ID you are trying to\n" +
				"register is invalid.\n" +
				"Please create a new license.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Die Profil-ID die du versuchst zu\n" +
				"registrieren ist ungültig.\n" +
				"Bitte erstelle ein neues Profil.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"El perfil que está tratando de\n" +
				"registrar es invalido.\n" +
				"Cree una nueva licencia.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"L'ID del profilo che stai cercando\n" +
				"di registrare non è valido.\n" +
				"Crea una nuova patente e riprova.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Het profiel-ID dat je probeert te\n" +
				"registreren is ongeldig.\n" +
				"Maak een nieuw profiel aan.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"L'ID du profil que vous essayez\n" +
				"d'enregistrer est invalide.\n" +
				"Veuillez créer une nouveau permis.\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgProfileIDInUse = WWFCErrorMessage{
		ErrorCode: 22007,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"あなたが登録しようとした \n" +
				"フレンドコードは すでに登録されています\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"The friend code you are trying to\n" +
				"register is already in use.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Den Freundescode den du gerade\n" +
				"registrierst wird bereits verwendet.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"La clave de amigo que está tratando\n" +
				"de registrar, ya está en uso.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Il codice amico che stai cercando\n" +
				"di registrare è già in uso.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"De vriendcode die je probeert te\n" +
				"registreren is al in gebruik.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Le code ami que vous essayez\n" +
				"d'enregistrer est déjà utilisé.\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgPayloadInvalid = WWFCErrorMessage{
		ErrorCode: 22008,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"Retro WFCの ペイロードがむこうです \n" +
				"ゲームを 再起動してください\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"The Retro WFC payload is invalid.\n" +
				"Try restarting your game.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Der Retro WFC payload ist ungültig.\n" +
				"Versuche das Spiel neu zu starten.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Retro WFC no cargó correctamente\n" +
				"Intente reiniciar su juego.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Il payload della Retro WFC non è valido.\n" +
				"Prova a riavviare il gioco.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"De Retro WFC-payload is ongeldig.\n" +
				"Probeer het spel opnieuw op te starten.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Le payload Retro WFC est invalide.\n" +
				"Veuillez redémarrer votre jeu.\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

	WWFCMsgInvalidELO = WWFCErrorMessage{
		ErrorCode: 22009,
		MessageRMC: map[byte]string{
			LangJapanese: "" +
				"VRまたはBRの値が むこうなため \n" +
				"Retro WFCから 切断されました\n" +
				"\n" +
				"エラーコード： %[1]d",
			LangEnglish: "" +
				"You were disconnected from\n" +
				"Retro WFC due to an invalid\n" +
				"VR or BR value.\n" +
				"\n" +
				"Error Code: %[1]d",
			LangGerman: "" +
				"Deine Verbindung zu Retro WFC\n" +
				"durch einen ungültigen VR oder BR\n" +
				"Wert beendet.\n" +
				"\n" +
				"Fehlercode: %[1]d",
			LangSpanish: "" +
				"Fuiste desconectado debido a discrepancias\n" +
				"con tu valor de PC o PB.\n" +
				"\n" +
				"Código de error: %[1]d",
			LangItalian: "" +
				"Sei stato disconnesso dalla Retro WFC\n" +
				"a causa di un valore non valido\n" +
				"di punti corsa o punti battaglia.\n" +
				"\n" +
				"Codice Errore: %[1]d",
			LangDutch: "" +
				"Je verbinding met Retro WFC is verbroken\n" +
				"vanwege een ongeldige rp- of gp-waarde.\n" +
				"\n" +
				"Foutcode: %[1]d",
			LangFrenchEU: "" +
				"Vous avez été déconnecté de\n" +
				"Retro WFC à cause d'une valeur invalide\n" +
				"de Points Course ou Points Bataille\n" +
				"\n" +
				"Code Erreur: %[1]d",
		},
	}

)
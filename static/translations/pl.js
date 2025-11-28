// Polish translations for GoRetro
window.translations = window.translations || {};

window.translations.pl = {
    // Nawigacja
    nav: {
        title: "GoRetro",
        user: "Użytkownik"
    },
    
    // Strona główna
    index: {
        pageTitle: "GoRetro - Narzędzie do Retrospektyw",
        createRoom: {
            title: "Utwórz Nową Retrospektywę",
            roomNameLabel: "Nazwa Pokoju",
            roomNamePlaceholder: "Retrospektywa",
            votesLabel: "Głosy na Użytkownika",
            createButton: "Utwórz Pokój"
        },
        myRooms: {
            title: "Moje Retrospektywy",
            noRooms: "Brak retrospektyw. Utwórz pierwszą, aby rozpocząć!",
            ownerBadge: "Właściciel"
        },
        howItWorks: {
            title: "Jak To Działa",
            step1: {
                title: "Tworzenie Notatek",
                description: "Uczestnicy dodają swoje przemyślenia i opinie"
            },
            step2: {
                title: "Łączenie",
                description: "Moderatorzy grupują podobne notatki"
            },
            step3: {
                title: "Głosowanie",
                description: "Wszyscy głosują na tematy do omówienia"
            },
            step4: {
                title: "Dyskusja",
                description: "Omówienie najważniejszych punktów i tworzenie zadań"
            },
            step5: {
                title: "Podsumowanie",
                description: "Przegląd opinii i zadań do wykonania"
            }
        }
    },
    
    // Strona pokoju
    room: {
        pageTitle: "{roomName} - GoRetro",
        shareLink: "Link do udostępnienia:",
        linkCopied: "Link do pokoju skopiowany do schowka!",
        linkCopyFailed: "Nie udało się skopiować linku",
        connectionStatus: {
            connecting: "Łączenie...",
            connected: "Połączono",
            disconnected: "Rozłączono",
            error: "Błąd",
            reconnecting: "Ponowne łączenie... ({attempt}/{max})",
            failed: "Połączenie nie powiodło się"
        },
        pendingApproval: {
            title: "Oczekiwanie na Zatwierdzenie",
            message: "Twoje żądanie dołączenia do pokoju oczekuje na zatwierdzenie przez moderatora lub właściciela.",
            pleaseWait: "Proszę czekać, Twój dostęp jest sprawdzany."
        },
        phases: {
            ticketing: "Notatki",
            merging: "Łączenie",
            voting: "Głosowanie",
            discussion: "Dyskusja",
            summary: "Podsumowanie"
        },
        votes: {
            info: "Wykorzystane głosy: {used} / {total}"
        },
        tickets: {
            title: "Notatki",
            addButton: "+ Dodaj Notatkę",
            placeholder: "Co Ci chodzi po głowie?",
            cancel: "Anuluj",
            submit: "Wyślij",
            noTickets: "Brak notatek. Bądź pierwszy, który doda!",
            votes: "{count} głosów",
            coveredBadge: "Omówione",
            markCovered: "Oznacz jako omówione",
            markNotCovered: "Oznacz jako nieomówione",
            unmergeAll: "Rozdziel wszystkie",
            separateFromParent: "Oddziel od głównej",
            deleteConfirmTitle: "Usuń Notatkę",
            deleteConfirmMessage: "Czy na pewno chcesz usunąć tę notatkę? Tej operacji nie można cofnąć.",
            delete: "Usuń",
            autoMerge: "🤖 Auto-łączenie",
            autoMerging: "⏳ Analizuję...",
            autoMergeConfirm: "Użyć AI do automatycznego grupowania podobnych notatek? System przeanalizuje treść i zasugeruje połączenia.",
            autoMergeComplete: "Auto-łączenie zakończone! {count} notatek zostało zgrupowanych."
        },
        actions: {
            title: "Zadania do Wykonania",
            addButton: "+ Dodaj Zadanie",
            placeholder: "Opisz zadanie do wykonania...",
            assignLabel: "Przypisz do (kliknij, aby zaznaczyć/odznaczyć):",
            selectAll: "Zaznacz Wszystkie",
            deselectAll: "Odznacz Wszystkie",
            assignedTo: "Przypisane do: {names}",
            noActions: "Brak zadań do wykonania.",
            cancel: "Anuluj",
            add: "Dodaj Zadanie",
            deleteConfirm: "Czy na pewno chcesz usunąć to zadanie?",
            autoPropose: "🤖 Auto-propozycje",
            autoProposing: "⏳ Analizuję...",
            teamContextLabel: "Kontekst zespołu (opcjonalnie):",
            teamContextPlaceholder: "Dodaj kontekst o zespole, stosie technologicznym lub ograniczeniach...",
            teamContextHelp: "To pomoże AI zasugerować bardziej trafne działania dla Twojej sytuacji.",
            generate: "Generuj Zadania",
            autoProposeComplete: "AI zasugerowało {count} zadań!"
        },
        participants: {
            title: "Uczestnicy",
            owner: "właściciel",
            moderator: "moderator",
            participant: "uczestnik",
            pending: "oczekujący",
            pendingApprovalTitle: "Oczekujące na Zatwierdzenie",
            promote: "↑ Awansuj",
            demote: "↓ Degraduj",
            remove: "✗ Usuń",
            approve: "✓ Zatwierdź",
            reject: "✗ Odrzuć",
            promoteToModerator: "Awansuj na moderatora",
            demoteFromModerator: "Degraduj z moderatora",
            removeFromRoom: "Usuń z pokoju",
            removeConfirm: "Czy na pewno chcesz usunąć tego uczestnika z pokoju?",
            rejectConfirm: "Czy na pewno chcesz odrzucić tego uczestnika?",
            noParticipants: "Brak uczestników",
            autoApproveLabel: "Automatyczne zatwierdzanie uczestników",
            autoApproveHelp: "Nowi uczestnicy dołączają automatycznie"
        },
        messages: {
            cannotPerformDisconnected: "Nie można wykonać akcji: brak połączenia z serwerem",
            unableToReconnect: "Nie można ponownie połączyć. Odśwież stronę.",
            accessApproved: "Twój dostęp został zatwierdzony!",
            requestRejected: "Twoje żądanie dołączenia zostało odrzucone"
        }
    },
    
    // Wspólne
    common: {
        cancel: "Anuluj",
        submit: "Wyślij",
        delete: "Usuń",
        close: "Zamknij",
        save: "Zapisz"
    }
};

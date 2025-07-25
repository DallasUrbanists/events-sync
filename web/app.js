function eventManager() {
    return {
        events: [],
        filteredEvents: [],
        stats: {},
        loading: false,
        statusFilter: 'pending',
        hideSingleEvents: false,

        async loadEvents() {
            this.loading = true;
            try {
                const response = await fetch('/api/events');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.events = await response.json();
                this.filterEvents();
            } catch (error) {
                console.error('Error loading events:', error);
                alert('Failed to load events. Please try again.');
            } finally {
                this.loading = false;
            }
        },

        async loadStats() {
            try {
                const response = await fetch('/api/events/stats');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.stats = await response.json();
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        },

        filterEvents() {
            let filtered = this.events;

            // Apply status filter (default to pending)
            if (this.statusFilter) {
                filtered = filtered.map(dateGroup => ({
                    ...dateGroup,
                    events: dateGroup.events.filter(event => event.review_status === this.statusFilter)
                })).filter(dateGroup => dateGroup.events.length > 0);
            }

            // Apply single event filter
            if (this.hideSingleEvents) {
                filtered = filtered.filter(dateGroup => dateGroup.events.length > 1);
            }

            this.filteredEvents = filtered;
        },

        async updateEventStatus(uid) {
            const selectElement = document.getElementById(`status-${uid}`);
            const newStatus = selectElement.value;

            try {
                const response = await fetch(`/api/events/${uid}/status`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ status: newStatus })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in our local data
                this.events.forEach(dateGroup => {
                    dateGroup.events.forEach(event => {
                        if (event.uid === uid) {
                            event.review_status = newStatus;
                        }
                    });
                });

                // Re-filter and reload stats
                this.filterEvents();
                this.loadStats();

                // Show success message
                this.showNotification('Status updated successfully!', 'success');
            } catch (error) {
                console.error('Error updating status:', error);
                this.showNotification('Failed to update status. Please try again.', 'error');
            }
        },

        showNotification(message, type = 'info') {
            // Create notification element
            const notification = document.createElement('div');
            notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all duration-300 ${
                type === 'success' ? 'bg-green-500 text-white' :
                type === 'error' ? 'bg-red-500 text-white' :
                'bg-blue-500 text-white'
            }`;
            notification.textContent = message;

            // Add to page
            document.body.appendChild(notification);

            // Remove after 3 seconds
            setTimeout(() => {
                notification.style.opacity = '0';
                setTimeout(() => {
                    if (notification.parentNode) {
                        notification.parentNode.removeChild(notification);
                    }
                }, 300);
            }, 3000);
        },

        formatDate(dateString) {
            const date = new Date(dateString);
            const today = new Date();
            const tomorrow = new Date(today);
            tomorrow.setDate(tomorrow.getDate() + 1);

            if (date.toDateString() === today.toDateString()) {
                return 'Today';
            } else if (date.toDateString() === tomorrow.toDateString()) {
                return 'Tomorrow';
            } else {
                return date.toLocaleDateString('en-US', {
                    weekday: 'long',
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                });
            }
        },

        formatTime(timeString) {
            const date = new Date(timeString);
            return date.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true
            });
        },

        getStatusClass(status) {
            switch (status) {
                case 'pending':
                    return 'status-pending';
                case 'reviewed':
                    return 'status-reviewed';
                case 'rejected':
                    return 'status-rejected';
                default:
                    return 'bg-gray-100 text-gray-800 border-gray-200';
            }
        },

        get totalEvents() {
            return this.events.reduce((total, dateGroup) => total + dateGroup.events.length, 0);
        }
    };
}
function eventManager() {
    return {
        events: [],
        filteredEvents: [],
        groupedEvents: [],
        stats: {},
        loading: false,
        statusFilter: '',
        hideSingleEvents: false,

        async loadEvents() {
            this.loading = true;
            try {
                const response = await fetch('/api/events');
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                this.events = await response.json();

                // Initialize editing properties for each event
                this.events.forEach(event => {
                    event.editingOrganization = false;
                    event.editingType = false;
                    event.editingLocation = false;
                });

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

        groupEventsByDate(events) {
            const eventsByDate = {};

            for (const event of events) {
                const date = new Date(event.start_time).toISOString().split('T')[0];
                if (!eventsByDate[date]) {
                    eventsByDate[date] = [];
                }
                eventsByDate[date].push(event);
            }

            // Convert to array and sort by date proximity to current date
            const result = [];
            for (const [date, dateEvents] of Object.entries(eventsByDate)) {
                result.push({
                    date: date,
                    events: dateEvents
                });
            }

            // Sort by date proximity (closest to current date first)
            const nowDate = new Date().toISOString().split('T')[0];
            result.sort((a, b) => {
                const dateA = new Date(a.date);
                const dateB = new Date(b.date);
                const now = new Date(nowDate);

                const diffA = Math.abs((dateA - now) / (1000 * 60 * 60 * 24));
                const diffB = Math.abs((dateB - now) / (1000 * 60 * 60 * 24));

                return diffA - diffB;
            });

            return result;
        },

        filterEvents() {
            let filtered = this.events;

            // Apply status filter
            if (this.statusFilter === 'approved') {
                filtered = filtered.filter(event => !event.rejected);
            } else if (this.statusFilter === 'rejected') {
                filtered = filtered.filter(event => event.rejected);
            }

            // Group events by date
            this.groupedEvents = this.groupEventsByDate(filtered);

            // Apply single event filter
            if (this.hideSingleEvents) {
                this.groupedEvents = this.groupedEvents.filter(dateGroup => dateGroup.events.length > 1);
            }

            this.filteredEvents = this.groupedEvents;
        },

        async updateEventStatus(uid, recurrenceID, approved) {
            const rejected = !approved;

            try {
                const response = await fetch(`/api/events/${uid}`, {
                    method: 'PATCH',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        recurrence_id: recurrenceID || '',
                        rejected: rejected
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in our local data
                this.events.forEach(event => {
                    if (event.uid === uid && (event.recurrence_id || '') === (recurrenceID || '')) {
                        event.rejected = rejected;
                    }
                });

                // Re-filter and reload stats
                this.filterEvents();
                this.loadStats();

                // Show success message
                this.showNotification('Event status updated successfully!', 'success');
            } catch (error) {
                console.error('Error updating status:', error);
                this.showNotification('Failed to update event status. Please try again.', 'error');
            }
        },

        async updateEventOrganization(uid, recurrenceID, organization) {
            try {
                const response = await fetch(`/api/events/${uid}`, {
                    method: 'PATCH',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        recurrence_id: recurrenceID || '',
                        organization: organization
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in our local data
                this.events.forEach(event => {
                    if (event.uid === uid && (event.recurrence_id || '') === (recurrenceID || '')) {
                        event.organization = organization;
                    }
                });

                // Re-filter
                this.filterEvents();

                // Show success message
                this.showNotification('Event organization updated successfully!', 'success');
            } catch (error) {
                console.error('Error updating organization:', error);
                this.showNotification('Failed to update event organization. Please try again.', 'error');
            }
        },

        async saveOrganizationChange(event) {
            // Save the change and exit edit mode
            await this.updateEventOrganization(event.uid, event.recurrence_id || '', event.organization);
            event.editingOrganization = false;
        },

        cancelOrganizationChange(event) {
            // Revert to original value and exit edit mode
            event.organization = event.originalOrganization;
            event.editingOrganization = false;
        },

        async updateEventType(uid, recurrenceID, eventType) {
            try {
                const response = await fetch(`/api/events/${uid}`, {
                    method: 'PATCH',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        recurrence_id: recurrenceID || '',
                        type: eventType
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update ALL events with the same UID (matching backend behavior)
                this.events.forEach(event => {
                    if (event.uid === uid) {
                        event.type = eventType;
                    }
                });

                // Re-filter
                this.filterEvents();

                // Show success message
                this.showNotification('Event type updated successfully!', 'success');
            } catch (error) {
                console.error('Error updating event type:', error);
                this.showNotification('Failed to update event type. Please try again.', 'error');
            }
        },

        async saveTypeChange(event) {
            // Save the change and exit edit mode
            await this.updateEventType(event.uid, event.recurrence_id || '', event.type);
            event.editingType = false;
        },

        cancelTypeChange(event) {
            // Revert to original value and exit edit mode
            event.type = event.originalType;
            event.editingType = false;
        },

        async setLocationOverlay(uid, recurrenceID, location) {
            try {
                const response = await fetch(`/api/events/${uid}/overlay`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        field: 'location',
                        value: location,
                        mergeLogic: 'overwrite_empty',
                        reason: 'Manual location override for Meetup events'
                    })
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in the local array
                this.events.forEach(event => {
                    if (event.uid === uid && (event.recurrence_id || '') === (recurrenceID || '')) {
                        if (!event.overlay) {
                            event.overlay = {};
                        }
                        event.overlay.location = {
                            value: location,
                            mergeLogic: 'overwrite_empty',
                            source: 'manual',
                            timestamp: new Date().toISOString(),
                            reason: 'Manual location override for Meetup events'
                        };
                    }
                });

                // Re-filter
                this.filterEvents();

                // Show success message
                this.showNotification('Location overlay set successfully!', 'success');
            } catch (error) {
                console.error('Error setting location overlay:', error);
                this.showNotification('Failed to set location overlay. Please try again.', 'error');
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
            const parts = dateString.split("-");
            const date = new Date(parseInt(parts[0]), parseInt(parts[1]) - 1, parseInt(parts[2]));
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

        async removeLocationOverlay(uid, recurrenceID) {
            try {
                const response = await fetch(`/api/events/${uid}/overlay/location`, {
                    method: 'DELETE'
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                // Update the event in the local array
                this.events.forEach(event => {
                    if (event.uid === uid && (event.recurrence_id || '') === (recurrenceID || '')) {
                        if (event.overlay && event.overlay.location) {
                            delete event.overlay.location;
                        }
                    }
                });

                // Re-filter
                this.filterEvents();

                // Show success message
                this.showNotification('Location overlay removed successfully!', 'success');
            } catch (error) {
                console.error('Error removing location overlay:', error);
                this.showNotification('Failed to remove location overlay. Please try again.', 'error');
            }
        },

        async saveLocationChange(event) {
            const newLocation = event.editingLocationValue ? event.editingLocationValue.trim() : '';

            if (newLocation) {
                // Set location overlay
                await this.setLocationOverlay(event.uid, event.recurrence_id || '', newLocation);
            } else if (event.overlay?.location) {
                // Remove existing overlay if field is empty
                await this.removeLocationOverlay(event.uid, event.recurrence_id || '');
            }

            event.editingLocation = false;
        },

        cancelLocationChange(event) {
            // Revert to original value and exit edit mode
            event.editingLocationValue = event.originalLocation;
            event.editingLocation = false;
        },

        get totalEvents() {
            return this.events.length;
        }
    };
}
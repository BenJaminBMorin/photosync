import SwiftUI

struct FilterOptionsView: View {
    @ObservedObject var viewModel: GalleryViewModel
    @Environment(\.dismiss) var dismiss

    var body: some View {
        NavigationStack {
            List {
                Section("Display Options") {
                    Toggle(isOn: $viewModel.showUnsyncedOnly) {
                        Label("Unsynced Only", systemImage: "icloud.slash")
                    }

                    Toggle(isOn: $viewModel.showIgnoredPhotos) {
                        Label("Show Ignored", systemImage: "eye.slash")
                    }

                    Toggle(isOn: $viewModel.showHiddenPhotos) {
                        Label("Show Hidden", systemImage: "eye.trianglebadge.exclamationmark")
                    }

                    Toggle(isOn: $viewModel.showServerOnlyPhotos) {
                        Label("Show From Server", systemImage: "cloud")
                    }
                }

                Section {
                    Toggle(isOn: $viewModel.enableDateFilter) {
                        Label("Filter by Date", systemImage: "calendar")
                    }

                    if viewModel.enableDateFilter {
                        DatePicker(
                            "From",
                            selection: $viewModel.dateFilterStart,
                            displayedComponents: .date
                        )

                        DatePicker(
                            "To",
                            selection: $viewModel.dateFilterEnd,
                            in: viewModel.dateFilterStart...,
                            displayedComponents: .date
                        )
                    }
                } header: {
                    Text("Date Range")
                } footer: {
                    if viewModel.enableDateFilter {
                        Text("Showing photos from \(viewModel.dateFilterStart.formatted(date: .abbreviated, time: .omitted)) to \(viewModel.dateFilterEnd.formatted(date: .abbreviated, time: .omitted))")
                    }
                }

                Section("Quick Filters") {
                    Button {
                        viewModel.setDateFilter(.today)
                    } label: {
                        Label("Today", systemImage: "calendar.badge.clock")
                    }

                    Button {
                        viewModel.setDateFilter(.thisWeek)
                    } label: {
                        Label("This Week", systemImage: "calendar.badge.clock")
                    }

                    Button {
                        viewModel.setDateFilter(.thisMonth)
                    } label: {
                        Label("This Month", systemImage: "calendar.badge.clock")
                    }

                    Button {
                        viewModel.setDateFilter(.thisYear)
                    } label: {
                        Label("This Year", systemImage: "calendar.badge.clock")
                    }
                }

                Section {
                    Button(role: .destructive) {
                        viewModel.resetFilters()
                    } label: {
                        Label("Reset All Filters", systemImage: "arrow.counterclockwise")
                    }
                }
            }
            .navigationTitle("Filter Options")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
    }
}
